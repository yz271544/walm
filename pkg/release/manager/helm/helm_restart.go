package helm

import (
	"WarpCloud/walm/pkg/k8s/adaptor"
	"WarpCloud/walm/pkg/k8s/handler"
	"github.com/sirupsen/logrus"
	"fmt"
	"sync"
)

func (hc *HelmClient) RestartRelease(namespace, releaseName string) error {
	logrus.Debugf("Enter RestartRelease %s %s\n", namespace, releaseName)
	releaseInfo, err := hc.GetRelease(namespace, releaseName)
	if err != nil {
		logrus.Errorf("failed to get release info : %s", err.Error())
		return err
	}

	podsToRestart := releaseInfo.Status.GetPodsNeedRestart()
	podsRestartFailed := []string{}
	mux := &sync.Mutex{}
	var wg sync.WaitGroup
	for _, podToRestart := range podsToRestart {
		wg.Add(1)
		go func(podToRestart *adaptor.WalmPod) {
			defer wg.Done()
			err1 := handler.GetDefaultHandlerSet().GetPodHandler().DeletePod(podToRestart.Namespace, podToRestart.Name)
			if err1 != nil {
				logrus.Errorf("failed to restart pod %s/%s : %s", podToRestart.Namespace, podToRestart.Name, err1.Error())
				mux.Lock()
				podsRestartFailed = append(podsRestartFailed, podToRestart.Namespace+"/"+podToRestart.Name)
				mux.Unlock()
				return
			}
		}(podToRestart)
	}

	wg.Wait()
	if len(podsRestartFailed) > 0 {
		err = fmt.Errorf("failed to restart pods : %v", podsRestartFailed)
		logrus.Errorf("failed to restart release : %s", err.Error())
		return err
	}

	logrus.Infof("succeed to restart release %s", releaseName)
	return nil
}