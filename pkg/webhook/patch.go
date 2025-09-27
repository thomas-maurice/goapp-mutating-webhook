package webhook

import (
	"encoding/json"

	jsonpatchv2 "gomodules.xyz/jsonpatch/v2"
)

// ComputePatch computes the json patch between two objects
// so that it can be returned to the apiserver for mutation.
func ComputePatch(old, new any) ([]byte, error) {
	oldBytes, err := json.Marshal(old)
	if err != nil {
		return nil, err
	}

	newBytes, err := json.Marshal(new)
	if err != nil {
		return nil, err
	}

	patch, err := jsonpatchv2.CreatePatch(oldBytes, newBytes)
	if err != nil {
		return nil, err
	}

	patchBytes, err := json.MarshalIndent(patch, "", "  ")
	if err != nil {
		return nil, err
	}

	return patchBytes, nil
}
