package mutator

import (
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CheckRequest will check the incomming request
// namely this will make sure we are indeed dealing with
// a pod object and not some other random things
func CheckRequest(request *admissionv1.AdmissionReview) error {
	if request == nil || request.Request == nil {
		return fmt.Errorf("invalid request: nil request")
	}

	// Are we actually working on a pod object ?
	podType := metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	if request.Request.Resource != podType {
		return fmt.Errorf("invalid resource type sent to the mutating endpoint: %v", request.Request.Resource)
	}

	return nil
}
