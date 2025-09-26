package mutator

import (
	"fmt"
	"io"

	"github.com/gin-gonic/gin"
	admissionv1 "k8s.io/api/admission/v1"
)

// DecodeAdmissionReview will parse the incoming request.
func DecodeAdmissionReview(ctx *gin.Context) (*admissionv1.AdmissionReview, error) {
	if ctx.Request.Header.Get("Content-Type") != "application/json" {
		return nil, fmt.Errorf("expected application/json content-type")
	}

	var body []byte
	if ctx.Request.Body != nil {
		requestData, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			return nil, err
		}
		body = requestData
	}

	admissionReviewRequest := &admissionv1.AdmissionReview{}
	if _, _, err := deserializer.Decode(body, nil, admissionReviewRequest); err != nil {
		return nil, err
	}

	return admissionReviewRequest, nil
}
