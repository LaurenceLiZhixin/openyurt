package site

import (
	"fmt"
	"github.com/openyurtio/openyurt/pkg/yurthub/cachemanager"
	"github.com/openyurtio/openyurt/pkg/yurthub/proxy/remote"
	"github.com/openyurtio/openyurt/pkg/yurthub/storage"
	"github.com/openyurtio/openyurt/pkg/yurthub/util"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"net/http"
	"sync"
)

type watchManager struct {
	storage cachemanager.StorageWrapper
	lb      remote.LoadBalancer

	watchMap     map[string]*watchCtx
	watchMapLock sync.RWMutex
}

func newWatchManager(storagewrapper cachemanager.StorageWrapper, lb remote.LoadBalancer) *watchManager {
	return &watchManager{
		storage:  storagewrapper,
		lb:       lb,
		watchMap: make(map[string]*watchCtx),
	}
}

func (w *watchManager) IsWatching(req *http.Request) (string, bool, error) {
	ctx := req.Context()

	info, ok := apirequest.RequestInfoFrom(ctx)
	if !ok || info == nil {
		return "", false, fmt.Errorf("request info is empty")
	}

	comp, ok := util.ClientComponentFrom(ctx)
	if !ok || len(comp) == 0 {
		comp = "default"
	}

	storageKey, err := w.storage.KeyFunc(storage.KeyBuildInfo{
		Component: comp,
		Resources: info.Resource,
		Namespace: info.Namespace,
		Name:      info.Name,
		Group:     info.APIGroup,
		Version:   info.APIVersion,
	})
	if err != nil {
		return "", false, err
	}
	w.watchMapLock.Lock()
	defer w.watchMapLock.Unlock()
	return storageKey.Key(), w.watchMap[storageKey.Key()] != nil, nil
}

func (w *watchManager) Handle(rw http.ResponseWriter, req *http.Request, addCurrentRequest bool) error {
	ctx := req.Context()
	info, ok := apirequest.RequestInfoFrom(ctx)
	if !ok || info == nil {
		return fmt.Errorf("request info is empty")
	}

	storageKey, _, err := w.IsWatching(req)
	if err != nil {
		return err
	}
	// get or create ctx
	w.watchMapLock.Lock()
	targetWatchCtx := w.watchMap[storageKey]
	if targetWatchCtx == nil {
		targetWatchCtx = newWatchCtx(info, w.lb)
		go func() {
			if err := targetWatchCtx.runWatch(); err != nil {
				panic(err)
			}
		}()
		w.watchMap[storageKey] = targetWatchCtx
	}
	w.watchMapLock.Unlock()

	if addCurrentRequest {
		// current watch subscribe
		targetWatchCtx.RunSubscribe(rw, ctx)
	}
	return nil
}
