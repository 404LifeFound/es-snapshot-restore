package k8s

import (
	"fmt"
	"strings"

	"github.com/404LifeFound/es-snapshot-restore/config"
	commonv1 "github.com/elastic/cloud-on-k8s/v3/pkg/apis/common/v1"
	esv1 "github.com/elastic/cloud-on-k8s/v3/pkg/apis/elasticsearch/v1"
	eslabel "github.com/elastic/cloud-on-k8s/v3/pkg/controller/elasticsearch/label"
	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/clientcmd"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Client struct {
	runtimeclient.Client
}

func NewClient() (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", config.GlobalConfig.Kube.Config)
	if err != nil {
		log.Error().Err(err).Msg("failed to build kubernetes config")
		return nil, err
	}

	scheme := runtime.NewScheme()
	esv1.AddToScheme(scheme)

	c, err := runtimeclient.New(
		config,
		runtimeclient.Options{
			Scheme: scheme,
		},
	)

	if err != nil {
		log.Error().Err(err).Msg("faild to create controller runtime client")
		return nil, err
	}

	return &Client{Client: c}, nil
}

type ESNodeSetOption func(*ESNodeSet)

type ESNodeSet struct {
	NodeSet *esv1.NodeSet
}

func NewESNodeSet(name, size string) *ESNodeSet {
	var tolerations []v1.Toleration
	for k, v := range config.GlobalConfig.ES.Tolerations {
		tolerations = append(tolerations, v1.Toleration{
			Key:      k,
			Operator: v1.TolerationOpEqual,
			Value:    v,
		})
	}

	labels := config.GlobalConfig.ES.Labels
	labels["app.kubernetes.io/instance"] = fmt.Sprintf("%s", name)

	return &ESNodeSet{
		NodeSet: &esv1.NodeSet{
			Name:  name,
			Count: config.GlobalConfig.ES.RestoreCount,
			Config: &commonv1.Config{
				Data: map[string]interface{}{
					fmt.Sprintf("node.attr.%s", config.GlobalConfig.ES.RestoreKey): name,
					"node.store.allow_mmap": false,
					"node.roles":            []string{"data"},
				},
			},
			PodTemplate: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Labels:      labels,
					Annotations: config.GlobalConfig.ES.Annotations,
				},
				Spec: v1.PodSpec{
					//TODO
					Affinity: &v1.Affinity{
						NodeAffinity: &config.GlobalConfig.ES.NodeAffinity,
						PodAntiAffinity: &v1.PodAntiAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
								{
									TopologyKey: config.GlobalConfig.ES.TopologyKey,
									LabelSelector: &metav1.LabelSelector{
										MatchLabels: map[string]string{
											eslabel.StatefulSetNameLabelName: fmt.Sprintf("%s-es-%s", config.GlobalConfig.ES.Name, name),
										},
									},
								},
							},
						},
					},
					Tolerations:        tolerations,
					ServiceAccountName: config.GlobalConfig.ES.ServiceAccount,
					InitContainers: []v1.Container{
						{
							Name: "install-plugins",
							Command: []string{
								"sh",
								"-c",
								fmt.Sprintf("bin/elasticsearch-plugin install --batch %s", strings.Join(config.GlobalConfig.ES.Plugins, " ")),
							},
						},
					},
					Containers: []v1.Container{
						{
							Name: config.GlobalConfig.ES.ContainerName,
							Resources: v1.ResourceRequirements{
								Limits: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse(config.GlobalConfig.ES.LimitCPU),
									v1.ResourceMemory: resource.MustParse(config.GlobalConfig.ES.LimitMem),
								},
								Requests: v1.ResourceList{
									v1.ResourceCPU:    resource.MustParse(config.GlobalConfig.ES.RequestCPU),
									v1.ResourceMemory: resource.MustParse(config.GlobalConfig.ES.RequestMem),
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []v1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "elasticsearch-data",
						Labels:      labels,
						Annotations: config.GlobalConfig.ES.Annotations,
					},
					Spec: v1.PersistentVolumeClaimSpec{
						AccessModes: []v1.PersistentVolumeAccessMode{
							v1.ReadWriteOnce,
						},
						Resources: v1.VolumeResourceRequirements{
							Requests: v1.ResourceList{
								v1.ResourceStorage: resource.MustParse(size),
							},
						},
						StorageClassName: ptrToString(config.GlobalConfig.ES.StorageClass),
					},
				},
			},
		},
	}
}
