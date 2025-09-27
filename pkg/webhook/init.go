package webhook

import (
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	defaultScheme "k8s.io/client-go/kubernetes/scheme"
)

var (
	scheme       *runtime.Scheme
	codecs       serializer.CodecFactory
	deserializer runtime.Decoder
)

func init() {
	scheme = defaultScheme.Scheme
	err := admissionv1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	/*err = corev1.AddToScheme(scheme)
	if err != nil {
		panic(err)
	}
	err = metav1.AddMetaToScheme(scheme)
	if err != nil {
		panic(err)
	}*/
	codecs = serializer.NewCodecFactory(scheme)
	deserializer = codecs.UniversalDeserializer()
}
