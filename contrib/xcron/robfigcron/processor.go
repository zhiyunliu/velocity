package robfigcron

import (
	sctx "context"
	"fmt"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/robfig/cron/v3"
	"github.com/zhiyunliu/glue/contrib/alloter"
	"github.com/zhiyunliu/glue/dlocker"
	"github.com/zhiyunliu/glue/log"
	"github.com/zhiyunliu/glue/standard"
	"github.com/zhiyunliu/glue/xcron"
	"github.com/zhiyunliu/golibs/xstack"
)

// processor cron管理程序，用于管理多个任务的执行，暂停，恢复，动态添加，移除
type processor struct {
	ctx             sctx.Context
	closeChan       chan struct{}
	onceLock        sync.Once
	jobs            cmap.ConcurrentMap
	monopolyJobs    cmap.ConcurrentMap
	routerEngine    *alloter.Engine
	immediatelyJobs []cron.FuncJob
	cronStdEngine   *cron.Cron
	cronSecEngine   *cron.Cron
}

type procJob struct {
	job     *xcron.Job
	entryid cron.EntryID
	engine  *cron.Cron
}

// NewProcessor 创建processor
func newProcessor(ctx sctx.Context, engine *alloter.Engine) (p *processor, err error) {
	p = &processor{
		ctx:           ctx,
		closeChan:     make(chan struct{}),
		jobs:          cmap.New(),
		monopolyJobs:  cmap.New(),
		routerEngine:  engine,
		cronStdEngine: cron.New(),
		cronSecEngine: cron.New(cron.WithSeconds()),
	}
	return p, nil
}

// Items Items
func (s *processor) Items() map[string]interface{} {
	return s.jobs.Items()
}

// Start 所有任务
func (s *processor) Start() error {
	s.handleImmediatelyJob()
	go s.cronStdEngine.Run()
	go s.cronSecEngine.Run()

	return nil
}

// Add 添加任务
func (s *processor) Add(jobs ...*xcron.Job) (err error) {
	var curEngine *cron.Cron
	for _, t := range jobs {
		if t.Disable {
			s.Remove(t.GetKey())
			continue
		}
		if err = t.Init(); err != nil {
			return
		}
		if err := s.checkIsMonopoly(t); err != nil {
			return err
		}
		curEngine = s.cronStdEngine
		if t.WithSeconds {
			curEngine = s.cronSecEngine
		}
		funcJob := s.buildFuncJob(t)
		if t.IsImmediately() {
			s.immediatelyJobs = append(s.immediatelyJobs, funcJob)
		}

		if jobId, err := curEngine.AddJob(t.Cron, funcJob); err != nil {
			err = fmt.Errorf("AddJob:%s,err:%+v", t.Cron, err)
			return err
		} else {
			s.jobs.Set(t.GetKey(), &procJob{
				job:     t,
				entryid: jobId,
				engine:  curEngine,
			})
		}
	}
	return
}

// Remove 移除服务
func (s *processor) Remove(key string) {
	if req, ok := s.jobs.Get(key); ok {
		procJob := req.(*procJob)
		procJob.job.Disable = true
		procJob.engine.Remove(procJob.entryid)
	}
	s.jobs.Remove(key)
}

// Close 退出
func (s *processor) Close() error {
	s.onceLock.Do(func() {
		close(s.closeChan)
		s.cronStdEngine.Stop()
		s.cronSecEngine.Stop()
		s.closeMonopolyJobs()
	})
	return nil
}

func (s *processor) checkIsMonopoly(j *xcron.Job) (err error) {
	if !j.IsMonopoly() {
		return nil
	}
	defer func() {
		if obj := recover(); obj != nil {
			err = fmt.Errorf("cron任务包含monopoly时需要提供dlocker的配置:%v", obj)
		}
	}()
	ins := standard.GetInstance(dlocker.TypeNode)
	sdlocker := ins.(dlocker.StandardLocker)
	s.monopolyJobs.Upsert(j.GetKey(), j, func(exist bool, valueInMap, newValue interface{}) interface{} {
		if exist {
			return valueInMap
		}
		return &monopolyJob{
			job:    j,
			locker: sdlocker.GetDLocker().Build(fmt.Sprintf("glue:cron:locker:%s", j.GetKey()), dlocker.WithData(j.GetLockData())),
			expire: 300, //默认300秒
		}
	})
	return nil
}

func (s *processor) reset(req *Request) (err error) {
	err = s.releaseMonopolyJob(req.job)
	req.reset()
	return
}

func (s *processor) releaseMonopolyJob(job *xcron.Job) (err error) {
	//根据执行后，重置下一次的独占时间
	if !job.IsMonopoly() {
		return
	}
	val, ok := s.monopolyJobs.Get(job.GetKey())
	if !ok {
		return
	}
	mjob := val.(*monopolyJob)
	nextSecs := mjob.job.CalcExpireSeconds()
	err = mjob.locker.Renewal(nextSecs)
	return
}

func (s *processor) renewalMonopolyJob(job *xcron.Job) (err error) {
	if !job.IsMonopoly() {
		return
	}
	val, ok := s.monopolyJobs.Get(job.GetKey())
	if !ok {
		return
	}
	mjob := val.(*monopolyJob)
	err = mjob.locker.Renewal(mjob.expire)
	return
}

func (s *processor) closeMonopolyJobs() {
	for item := range s.monopolyJobs.IterBuffered() {
		item.Val.(*monopolyJob).Close()
	}
	s.monopolyJobs.Clear()
}

func (s *processor) buildFuncJob(job *xcron.Job) cron.FuncJob {
	req := newRequest(job)
	return func() {
		req.ctx = s.ctx
		s.handle(req)
	}
}

func (s *processor) handle(req *Request) {
	//任务是否处理中，如果是，直接退出
	if !req.CanProc() {
		return
	}
	logger := log.New(log.WithSid(req.session))

	done := make(chan struct{})
	defer func() {
		if obj := recover(); obj != nil {
			logger.Panicf("cron.handle.cron:%s,service:%s, error:%+v. stack:%s", req.job.Cron, req.job.Service, obj, xstack.GetStack(1))
		}
		if err := s.reset(req); err != nil {
			logger.Errorf("cron.handle.cron:%s,service:%s, error:%+v. reset", req.job.Cron, req.job.Service, err)
		}
		close(done)
	}()

	hasMonopoly, err := req.Monopoly(s.monopolyJobs)
	if err != nil {
		logger.Errorf("cron.handle.cron.2:%s,service:%s, error:%+v", req.job.Cron, req.job.Service, err)
		return
	}
	if hasMonopoly {
		logger.Warnf("cron.handle.cron.3:%s,service:%s,meta:%+v=>monopoly.key=%s", req.job.Cron, req.job.Service, req.job.Meta, req.job.GetKey())
		return
	}
	go s.handleMonopolyJobExpire(logger, req.job, done)

	req.ctx = sctx.Background()
	resp := newResponse()
	err = s.routerEngine.HandleRequest(req, resp)
	if err != nil {
		panic(err)
	}
	resp.Flush()
}

func (s *processor) handleImmediatelyJob() {
	for i := range s.immediatelyJobs {
		go func(idx int) {
			s.immediatelyJobs[idx]()
		}(i)
	}
}

func (s *processor) handleMonopolyJobExpire(logger log.Logger, job *xcron.Job, done chan struct{}) {
	ticker := time.NewTicker(time.Minute)
	defer func() {
		if obj := recover(); obj != nil {
			logger.Panicf("cron.jobexpire:%s,service:%s,meta:%+v,recover:%s, stack:%s", job.Cron, job.Service, job.Meta, job.GetKey(), xstack.GetStack(1))
		}
		ticker.Stop()
	}()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
		}
		err := s.renewalMonopolyJob(job)
		if err != nil {
			logger.Errorf("cron.jobexpire:%s,service:%s,meta:%+v,renewal.key=%s", job.Cron, job.Service, job.Meta, job.GetKey())
		}
	}
}

type monopolyJob struct {
	job    *xcron.Job
	locker dlocker.DLocker
	expire int
}

func (j *monopolyJob) Acquire() (bool, error) {
	return j.locker.Acquire(j.expire)
}

func (j *monopolyJob) Renewal() {
	j.locker.Renewal(j.expire)
}

func (j *monopolyJob) Close() {
	j.locker.Release()
}
