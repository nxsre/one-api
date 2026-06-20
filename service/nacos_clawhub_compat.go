package service

import "github.com/songquanpeng/one-api/model"

// NacosAIClawHubSkillInfo returns (name, description, latestOnlineVersion) for a
// skill in the Nacos AI registry, for the ClawHub-compatible
// GET /nacos/clawhub/api/v1/skills/{slug} endpoint. Reuses the same version
// resolution as skill download (ResolveVersionForGet with empty label/version =
// latest online). Errors if the skill is missing or has no online version.
func NacosAIClawHubSkillInfo(namespace, skillName string) (name, description, version string, err error) {
	ns := NormalizeNacosNamespaceID(namespace)
	a, err := findArtifact(ns, model.NacosAIKindSkill, skillName)
	if err != nil {
		return "", "", "", err
	}
	vs, err := listVersions(a.Id)
	if err != nil {
		return "", "", "", err
	}
	pv, err := ResolveVersionForGet(a, vs, "", "")
	if err != nil {
		return "", "", "", err
	}
	return a.Name, a.Description, pv.Version, nil
}
