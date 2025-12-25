package k8s

func ptrToString(s string) *string {
	return &s
}

/*
func GetNodeAffinity() (*v1.NodeAffinity, error) {
	var node_affinity v1.NodeAffinity
	if config.GlobalConfig.ES.NodeAffinity != nil {
		b, err := yaml.Marshal(config.GlobalConfig.ES.NodeAffinity)
		if err != nil {
			log.Error().Err(err).Msgf("failed to Marshal %v", config.GlobalConfig.ES.NodeAffinity)
			return nil, err
		}

		if err := yaml.Unmarshal(b, &node_affinity); err != nil {
			log.Error().Err(err).Msgf("failed to unmarshal %v to v1.NodeAffinity", b)
			return nil, err
		}

		return &node_affinity, nil
	}

	return nil, fmt.Errorf("No nodeAffinity exist")
}
*/
