package site

import (
	"fmt"
	"github.com/openyurtio/openyurt/pkg/yurthub/cachemanager"
	"github.com/openyurtio/openyurt/pkg/yurthub/proxy/remote"
	"github.com/openyurtio/openyurt/pkg/yurthub/proxy/util"
	hubutil "github.com/openyurtio/openyurt/pkg/yurthub/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/klog/v2"
	"net/http"
)

type SiteProxy struct {
	localProxy   http.Handler
	lb           remote.LoadBalancer
	watchManager *watchManager
}

func (s *SiteProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	ctx := req.Context()
	if reqInfo, ok := apirequest.RequestInfoFrom(ctx); ok && reqInfo != nil && reqInfo.IsResourceRequest {
		klog.V(3).Infof("go into local proxy for request %s", hubutil.ReqString(req))
		switch reqInfo.Verb {
		case "watch":
			err = s.watchManager.Handle(w, req, true)
		case "create", "delete", "deletecollection":
			s.localProxy.ServeHTTP(w, req)
			return
		default:
			// list., get, update
			// 1. check if watching
			_, exist, checkIsWatchingErr := s.watchManager.IsWatching(req)
			if checkIsWatchingErr != nil {
				err = checkIsWatchingErr
				break
			}

			if !exist {
				// 2. start list-watch long link to make sure local proxy is updated
				err = s.watchManager.Handle(w, req, false)
				s.lb.ServeHTTP(w, req)
				return
			}

			// already exist watch gr, just serve with cache
			s.localProxy.ServeHTTP(w, req)
		}

		if err != nil {
			klog.Errorf("could not site proxy for %s, %v", hubutil.ReqString(req), err)
			util.Err(err, w, req)
		}
	} else {
		klog.Errorf("request(%s) is not supported when cluster is unhealthy", hubutil.ReqString(req))
		util.Err(apierrors.NewBadRequest(fmt.Sprintf("request(%s) is not supported when cluster is unhealthy", hubutil.ReqString(req))), w, req)
	}
}

func NewSiteProxy(storagewrapper cachemanager.StorageWrapper, localProxy http.Handler, lb remote.LoadBalancer) *SiteProxy {
	return &SiteProxy{
		localProxy:   localProxy,
		lb:           lb,
		watchManager: newWatchManager(storagewrapper, lb),
	}
}
