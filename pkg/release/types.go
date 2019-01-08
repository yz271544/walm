package release

import (
	"walm/pkg/k8s/adaptor"
	"k8s.io/helm/pkg/transwarp"
)

type ReleaseInfoList struct {
	Num   int            `json:"num" description:"release num"`
	Items []*ReleaseInfo `json:"items" description:"releases list"`
}

type ReleaseInfo struct {
	ReleaseSpec
	Ready   bool                     `json:"ready" description:"whether release is ready"`
	Message string                   `json:"message" description:"why release is not ready"`
	Status  *adaptor.WalmResourceSet `json:"release_status" description:"status of release"`
}

type ReleaseSpec struct {
	Name            string                 `json:"name" description:"name of the release"`
	RepoName        string                 `json:"repo_name" description:"chart name"`
	ConfigValues    map[string]interface{} `json:"config_values" description:"extra values added to the chart"`
	Version         int32                  `json:"version" description:"version of the release"`
	Namespace       string                 `json:"namespace" description:"namespace of release"`
	Dependencies    map[string]string      `json:"dependencies" description:"map of dependency chart name and release"`
	ChartName       string                 `json:"chart_name" description:"chart name"`
	ChartVersion    string                 `json:"chart_version" description:"chart version"`
	ChartAppVersion string                 `json:"chart_app_version" description:"jsonnet app version"`
	HelmValues
}

type ReleaseCache struct {
	ReleaseSpec
	ReleaseResourceMetas []ReleaseResourceMeta  `json:"release_resource_metas" description:"release resource metas"`
	ComputedValues        map[string]interface{} `json:"computed_values" description:"release computed values"`
}

type ReleaseResourceMeta struct {
	Kind      string `json:"kind" description:"resource kind"`
	Namespace string `json:"namespace" description:"resource namespace"`
	Name      string `json:"name" description:"resource name"`
}

type ChartValicationInfo struct {
	Name         string                 `json:"name" description:"name of the release"`
	ConfigValues map[string]interface{} `json:"config_values" description:"extra values added to the chart"`
	Version      int32                  `json:"version" description:"version of the release"`
	Namespace    string                 `json:"namespace" description:"namespace of release"`
	Dependencies map[string]string      `json:"dependencies" description:"map of dependency chart name and release"`
	ChartName    string                 `json:"chart_name" description:"chart name"`
	ChartVersion string                 `json:"chart_version" description:"chart version"`
	RenderStatus string                 `json:"render_status" description:"status of rending "`
	RenderResult map[string]string      `json:"render_result" description:"result of rending "`
	DryRunStatus string                 `json:"dry_run_status" description:"status of dry run "`
	DryRunResult map[string]string      `json:"dry_run_result" description:"result of dry run "`
	ErrorMessage string                 `json:"error_message" description:" error msg "`
}

type ReleaseRequest struct {
	Name                string                 `json:"name" description:"name of the release"`
	RepoName            string                 `json:"repo_name" description:"chart name"`
	ChartName           string                 `json:"chart_name" description:"chart name"`
	ChartVersion        string                 `json:"chart_version" description:"chart repo"`
	ConfigValues        map[string]interface{} `json:"config_values" description:"extra values added to the chart"`
	Dependencies        map[string]string      `json:"dependencies" description:"map of dependency chart name and release"`
	ReleasePrettyParams PrettyChartParams      `json:"release_pretty_params" description:"pretty chart params for market"`
}

type HelmExtraLabels struct {
	HelmLabels map[string]interface{} `json:"helmlabels"`
}

type HelmValues struct {
	HelmExtraLabels     *HelmExtraLabels         `json:"HelmExtraLabels"`
	AppHelmValues       *transwarp.AppHelmValues `json:"HelmAdditionalValues"`
	ReleasePrettyParams PrettyChartParams        `json:"release_pretty_params" description:"pretty chart params for market"`
}

type RepoInfo struct {
	TenantRepoName string `json:"repo_name"`
	TenantRepoURL  string `json:"repo_url"`
}

type RepoInfoList struct {
	Items []*RepoInfo `json:"items" description:"chart repo list"`
}

type ChartDependencyInfo struct {
	ChartName   string `json:"chart_name"`
	MaxVersion	float32 `json:"max_version"`
	MinVersion  float32 `json:"min_version"`
}

type ChartInfo struct {
	ChartName         string                `json:"chart_name"`
	ChartVersion      string                `json:"chart_version"`
	ChartDescription  string                `json:"chart_description"`
	ChartAppVersion   string                `json:"chart_appVersion"`
	ChartEngine       string                `json:"chart_engine"`
	DefaultValue      string                `json:"default_value" description:"default values.yaml defined by the chart"`
	DependencyCharts  []ChartDependencyInfo `json:"dependency_charts" description:"dependency chart name"`
	ChartPrettyParams PrettyChartParams     `json:"chart_pretty_params" description:"pretty chart params for market"`
}

type ChartInfoList struct {
	Items []*ChartInfo `json:"items" description:"chart list"`
}

type ReleaseConfigDeltaEventType string

const (
	CreateOrUpdate ReleaseConfigDeltaEventType = "CreateOrUpdate"
	Delete         ReleaseConfigDeltaEventType = "Delete"
)

type ReleaseConfigDeltaEvent struct {
	Type ReleaseConfigDeltaEventType `json:"type" description:"delta type: CreateOrUpdate, Delete"`
	Data ReleaseConfig               `json:"data" description:"release config data"`
}

type ReleaseConfig struct {
	AppName      string             `json:"app_name" description:"chart name"`
	Version      string             `json:"version" description:"chart version"`
	InstanceName string             `json:"instance_name" description:"release name"`
	ConfigSets   []ReleaseConfigSet `json:"configsets" description:"configsets"`
}

type ReleaseConfigSet struct {
	Name        string              `json:"name" description:"name"`
	CreatedBy   string              `json:"created_by" description:"created by"`
	ConfigItems []ReleaseConfigItem `json:"config_items" description:"config items"`
	Format      string              `json:"format" description:"format"`
}

type ReleaseConfigItem struct {
	Name  string                 `json:"name" description:"name"`
	Value map[string]interface{} `json:"value" description:"value"`
	Type  string                 `json:"type" description:"value"`
}

type DummyServiceConfig struct {
	Provides map[string]DummyServiceConfigImmediateValue `json:"provides" description:"dummy service provides"`
}

type DummyServiceConfigImmediateValue struct {
	ImmediateValue map[string]interface{} `json:"immediate_value" description:"dummy service immediate value"`
}

// Pretty Paramters
type ResourceStorageConfig struct {
	Name         string   `json:"name"`
	StorageType  string   `json:"type"`
	StorageClass string   `json:"storageClass"`
	Size         string   `json:"size"`
	AccessModes  []string `json:"accessModes"`
	AccessMode   string   `json:"accessMode"`
}

type ResourceConfig struct {
	CpuLimit            float64                 `json:"cpu_limit"`
	CpuRequest          float64                 `json:"cpu_request"`
	MemoryLimit         float64                 `json:"memory_limit"`
	MemoryRequest       float64                 `json:"memory_request"`
	GpuLimit            int                     `json:"gpu_limit"`
	GpuRequest          int                     `json:"gpu_request"`
	ResourceStorageList []ResourceStorageConfig `json:"storage"`
}

type BaseConfig struct {
	ValueName        string      `json:"variable" description:"variable name"`
	DefaultValue     interface{} `json:"default" description:"variable default value"`
	ValueDescription string      `json:"description" description:"variable description"`
	ValueType        string      `json:"type" description:"variable type"`
}

type RoleConfig struct {
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Replicas           int            `json:"replicas"`
	RoleBaseConfig     []*BaseConfig  `json:"baseConfig"`
	RoleResourceConfig ResourceConfig `json:"resouceConfig"`
}

type CommonConfig struct {
	Roles []RoleConfig `json:"roles"`
}

type PrettyChartParams struct {
	CommonConfig        CommonConfig  `json:"commonConfig"`
	TranswarpBaseConfig []*BaseConfig `json:"transwarpBundleConfig"`
	AdvanceConfig       []*BaseConfig `json:"advanceConfig"`
}

type TranswarpAppInfo struct {
	transwarp.AppDependency
	UserInputParams PrettyChartParams `json:"userInputParams"`
}
