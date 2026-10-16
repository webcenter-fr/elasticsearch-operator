package logstash

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	discovercrd "github.com/webcenter-fr/elasticsearch-operator/api/discover/v1"
	logstashcrd "github.com/webcenter-fr/elasticsearch-operator/api/logstash/v1"
	"github.com/webcenter-fr/elasticsearch-operator/api/shared"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestBuildLogstashSecretsExternalLogstash(t *testing.T) {
	expectedLabels := map[string]string{
		"discoverName":                    "test-logstash",
		discovercrd.LogstashAnnotationKey: "true",
		discovercrd.DiscoverAnnotationKey: "true",
	}
	expectedEnvAnnotations := map[string]string{
		discovercrd.LogstashAnnotationKey:                              "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-logstash",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "env",
	}
	expectedFileAnnotations := map[string]string{
		discovercrd.LogstashAnnotationKey:                              "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-logstash",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "file",
	}
	secretUserLogstash := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "external-user-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"user.key": []byte(`-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCqC7iHvcmK4O7d
hCvKkQ7qfKQSFnSMPhzREfPaRllUI0CpI7MgCUzC6BDjqjTzL+Ai3lfyW+kB0cAE
FJktBh36Le+ObFoP2wTYfSpEIMiD9lwYm6k1Lb2wc8vZlq4zlxkqXtJKs9KNCBCd
Rg/u1iwanxvvibmYCCLvHu+Syi9icUVjc5Owv5Wg+OCh3EI4Wvo9eUb7oE2WMxCh
UukhGf3a6viq1g4uQPh8NrbjB2USIl1rNSINOeg5da7+VZLwE7diRRmoOd6geT6o
sfCOlEKnC9JGXg+B/a0AukwABBBYDNsGpN7pNZpn0zjU9T07i9+14ePeyzGH+7yy
Bvj+WU4JAgMBAAECggEABxCzmxg8noOYCSYrt5/tUbraClJah1UxV1J6olAX+BH6
5swv7h3Uzahcw0jYKF6N3oUEfHJrLQMtCj5c2u0NI87yzCUeXNhJzEDvF6OREUKU
MwWMs0PyRLma97B2Dnixt/v2mtn73pY+MnqtwMuvS2/e9kXvqyxzXyIW7C9wZpOi
54y1SUwrJtce/UMtluStnZtgl8OGO8mDDH14edXojlb9cOy2MdkHEWqmcFI8UJ1I
kVh36cpp+k/TkIAYeE81IME/XYJ5uOvnIfoqVFU2Av9IfYduGQbpc1yJR6Wf34rN
AVbuciaJfA5THC4b5s7OmggqmiadltpcLAafSwRMvQKBgQDY45+PJaQjfNRAtDkW
bo0Qo+qtSLF5hl3CC0fc+hM3mhDM935raP2aoP7kvuZhjsZkQRM4YWQKpPnlBwpn
YHguSSAcAFjBTzNvMagfRH7TGxvjhzFqjxJhsxh6WMjoDF4YPufYGUWHnoIltpPZ
EkHvvIsS1HAdQaQkzeFPOncpfwKBgQDItaP99j2H24nkzfBe1MdgsE8fR47XIpDh
V3mcb2GOGomG/wJam/hZ9cGorwSn3RYakwx0CREFp95r08nPS5JgBIUn5BAxllBt
y7eGyJNhzQ+p2inz+L9ALxQsdBizC2hjD5ItKYjhX5Gli4KSDEQR7Zvzg4TwmXSM
Gmma8eT8dwKBgQCHvgap97v//et1x5YHJQ+iv4RsCZgR3/ekL509f0IxxXoIXkm/
/cjjUynUjmnv2wTt0BCwc7vCMbi5D6wCQU1WYqv2+nySF/iv+hsn3p2lkEznHUDf
WUX+4bFLOXAcU0k80XoXNNHgbGM2KUvlStj7qzw7f7vfg6qw4i3GgWf9DwKBgDjQ
8nNZ2WE0DHzwrzKpGSeNwVghcZYE+u5PTOWzwfHwIS5N0eTEcjFFGGUf5jl+pFsx
dT0DtAXd12C+u0orImx72xXHwa3H10y9dl55OP9ehSiX0Xh22ezZZuSOmM9WU153
cfHG6DCkVR5/t56QSlSo5pSwjlltl0hx0aNDBodRAoGAP2X1Uza3iq0tngKWFIeO
77AxiS6VUb9tjQtQpFF1hmVMMDBLySb15D23M0SzcSzKrtlKxoDSn1FzzETWdSNt
i+x83aUnRlsg3OGRByNVMbj8mJI7m3z/BhN0sUY4ABYE6e/8PLamBnnNe54dhyLs
jymHpXH28vD3JQuliElFEB0=
-----END PRIVATE KEY-----`),
			"user.crt": []byte(`-----BEGIN CERTIFICATE-----
MIIEKzCCAhOgAwIBAgIUVVzGc7Xc5MY1IikS1w08rtP/rywwDQYJKoZIhvcNAQEN
BQAwLTETMBEGA1UECgwKaW8uc3RyaW16aTEWMBQGA1UEAwwNY2xpZW50cy1jYSB2
MDAeFw0yNTA5MTcxNjExNTZaFw0zNTA5MTUxNjExNTZaMBAxDjAMBgNVBAMMBWFk
bWluMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqgu4h73JiuDu3YQr
ypEO6nykEhZ0jD4c0RHz2kZZVCNAqSOzIAlMwugQ46o08y/gIt5X8lvpAdHABBSZ
LQYd+i3vjmxaD9sE2H0qRCDIg/ZcGJupNS29sHPL2ZauM5cZKl7SSrPSjQgQnUYP
7tYsGp8b74m5mAgi7x7vksovYnFFY3OTsL+VoPjgodxCOFr6PXlG+6BNljMQoVLp
IRn92ur4qtYOLkD4fDa24wdlEiJdazUiDTnoOXWu/lWS8BO3YkUZqDneoHk+qLHw
jpRCpwvSRl4Pgf2tALpMAAQQWAzbBqTe6TWaZ9M41PU9O4vfteHj3ssxh/u8sgb4
/llOCQIDAQABo2AwXjAdBgNVHQ4EFgQUX/lsJNN0QdyvAP7xJYffv80tumQwDAYD
VR0TAQH/BAIwADAOBgNVHQ8BAf8EBAMCBaAwHwYDVR0jBBgwFoAU9xTZqBCn/CmW
aR8tVq7HIGxQpWQwDQYJKoZIhvcNAQENBQADggIBACiDVABQrm4KBlr6AXi1vvIm
9Yjo8sHLukfCCkIr0WIgH4jwKVPBOBYmAhVVml3D2SpnbhVVl0AzUfmk56UTTD5D
aXGeO7d+n9JWzsMC1lmbwce7d1jOhtFAbqFHKr1eF3sTXGkd2TfcJW28slAGk6Ae
2CRi7YWMKs9H6V4xlIe8cMkSmTvB3+RiyDa7LKk/w9vtMShyQr1WtW0AWX+JjuFn
ruWfSny4Nz+sixAqYjEFoddvjJWjRkdsgFWUdvnftIw9qmr8nrOISOnvMEfrtJ5c
xM9Ti53sFvO8bKKubDdJZZHIeSaLNmF1YCo3fb/oKzrpCIseTSCvVqeugn2B/4Mq
G/YtHB6XIQp9qWmBjiJTJ8bkyRz3t87XpKdfr/Y2Ryh1ABwfoSQgnkNsEDyufVIe
UmRSIftNQ1wAWVYN+U74/pqkq0ZBM6pelMEgaGqV7JhNtwY50XuFHoBFGKzIf1WY
AlDRJf5YBbM4SkWGIVXsSHWTkFJq6avkMLdeHEJZH7e5j16tNOI5ROwNALCwMmhz
uu9SSXvmLX73L0Kc8U5tCHd02bL4jBUrC2M0h1yVRfHH7t9a7s38lAsEOCMmEmhR
f4j8bexXdBM/6h6sPlV5nejz2fPb5lKoLS0tCg02n0WC6jNQF/yAdNFM0kzBxspj
S/PdFt4dZmYbx/dqWCV+
-----END CERTIFICATE-----`),
		},
	}
	secretCustomCa := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-ca-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte(`-----BEGIN CERTIFICATE-----
MIIFLTCCAxWgAwIBAgIUMA+B6fEMWAfNo6h9erM5ucd9a40wDQYJKoZIhvcNAQEN
BQAwLTETMBEGA1UECgwKaW8uc3RyaW16aTEWMBQGA1UEAwwNY2x1c3Rlci1jYSB2
MDAeFw0yNTA5MTcxNTUxNDdaFw0zNTA5MTUxNTUxNDdaMC0xEzARBgNVBAoMCmlv
LnN0cmltemkxFjAUBgNVBAMMDWNsdXN0ZXItY2EgdjAwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQDgUq+kfwF7fp2QVi30rpAOux9GXyJUQKmSCinW4XIJ
XLo0tjv/YzaWqNQdoXDiY3Q22nsJPE1KHKliNYTaO85VC2/PWZG9CMmfwPcoL7ES
vT1K3zTp7lBR7X3NDcseHyEIx4dCAi/Rs68FTPRjGKRVTfxOBqhuMJIVl+ztn2WJ
E2rnHTl2IpEZaE5fQ9CEfe6mcBEL5O0DnlI6By4r/n71dzkco17GnYDAuebz0ymT
vBs6ByhsZBVC/T8Pqo3eeN99Fj6EQCJnSwgFN1qth02dSrVYyV74HW1XhknjprzH
W7plW3RZOVG6HhLEP7edTnofZf0Dv0tcGtkq1kwNHdGzDW3yIHB0k4imAzYQvLUS
tWWItM1vlyOI92I4kDZZjZSFypfiQJ5HZBmXvJXdwLTMK7T+eDPRoO8UJD90OxPh
0CVcMCe1+BHQe6Qr2+i1pXwKyD+fGh1ggGWDMw0M5GxfK2CDeRTulDn8kgJiE6Fa
bCwkGAw9mon9gocSd6WUR5YK/hELe7H5P4BK+qD2CHt2ogmKE/prlx7tBLqmC4aH
CHUJyAzjMcRp6AMAtewXnyyStLOIVGFFCuHiZRYawx3qkaXMIQYG8zFQamMvR9Aa
EJCWkkaGxSyjatyiSXWMiXO+axm62HEDzF7YGKmVAQxOozqIcH+QcQaQHTtM4ZGE
NQIDAQABo0UwQzAdBgNVHQ4EFgQUvyIeTeFhnS6Od/Mr1ARA6EFx+94wEgYDVR0T
AQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAQYwDQYJKoZIhvcNAQENBQADggIB
AEvcZYOFFMv/t4b7Xsn0AfdBYBPm/qrvqSf30ZJU6dWMt6Iq99+/2w+ljx4JRjXc
xXj4L9l/5Riwo2XlPebabr9uIhLusY1k0aBIyUcClZM17/2A4ndkOjHLD3aRl5ho
b0P/l9xukcQ4giOVp06OG1p0xjy+QnE8Wa/Z2NI0plGyVMeDURl3QhvX/f3sZv0B
fPRNIbBeJCilV7GVNtoJ/zW+hLBX3daXx03Qubjn62DmUiDT6qP6LASdDTHNwTxE
pdVsuLweAfZ7kN65Pu89RIvwQjmlIepE1n73U/9jt8S+zGUS/nOc6I72UWTNRoC2
I1jw0jNefP0+wul39ekuSqzyyCYXRvzS6TFAjMp1ELJ2sCsz1ZOfMZVoP1wqgWH6
OkaKy9BpSGQCuhBAObUE1UPJMCbwwpq4yPCRHVuB0qRJ/fQ6vj+erXfm4CPG9N7k
bXg+WBGBKoTnSjJ1rwGA/8OgIScQOYMRx9txQE0xA7+kI6fb4A6HShX1SJH+1+XT
iYdyQURU58c9nE/CgpUhzmhbEoIC8DI3mu6LYRXBzIQdPSBZxLPG+eWcyebJ4Vfn
LsoolAK9LU9qB9Ad9qjzI6ywAcQt9Gs0KAslsVCapO2Gp/VFjjKrvG+SS+alQF9H
u/w9jG82luQgK3mBxGSVA/H+QYQIspbCrixnUhDlFUdP
-----END CERTIFICATE-----`),
		},
	}

	// Test with external Logstash configuration withtout specified target secrets
	logstash := &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			LogstashRef: discovercrd.LogstashRef{
				ExternalLogstashRef: &discovercrd.LogstashExternalRef{
					Addresses: []string{
						"logstash1.example.com:9092",
						"logstash2.example.com:9092",
					},
					UserSecretRef: &corev1.LocalObjectReference{
						Name: "external-user-secret",
					},
				},
			},
		},
	}
	secrets, err := buildLogstashSecrets(logstash, nil, nil, secretUserLogstash, nil)

	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Len(t, secrets, 2) // Should return both env and file secrets
	assert.Equal(t, "test-logstash-env", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, expectedLabels, secrets[0].Labels)
	assert.Equal(t, expectedEnvAnnotations, secrets[0].Annotations)
	assert.Equal(t, []byte("logstash1.example.com:9092,logstash2.example.com:9092"), secrets[0].Data["LOGSTASH_HOSTS_TEST_LOGSTASH"])
	assert.NotEmpty(t, secrets[0].Data["LOGSTASH_USER_PASSWORD_TEST_LOGSTASH"])
	assert.Equal(t, "test-logstash-file", secrets[1].Name)
	assert.Equal(t, "default", secrets[1].Namespace)
	assert.Equal(t, expectedLabels, secrets[1].Labels)
	assert.Equal(t, expectedFileAnnotations, secrets[1].Annotations)
	assert.Equal(t, string(secretUserLogstash.Data["user.crt"]), string(secrets[1].Data["user.crt"]))
	assert.Equal(t, string(secretUserLogstash.Data["user.key"]), string(secrets[1].Data["user.key"]))
	assert.NotEmpty(t, secrets[1].Data["user.p12"])

	// Test when user specified target secrets
	logstash = &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "logstash-env-secret",
				},
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "logstash-file-secret",
				},
			},
			LogstashRef: discovercrd.LogstashRef{
				ExternalLogstashRef: &discovercrd.LogstashExternalRef{
					Addresses: []string{
						"logstash1.example.com:9092",
						"logstash2.example.com:9092",
					},
					UserSecretRef: &corev1.LocalObjectReference{
						Name: "external-user-secret",
					},
				},
			},
		},
	}
	secrets, err = buildLogstashSecrets(logstash, nil, nil, secretUserLogstash, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Len(t, secrets, 2) // Should return both env and file secrets
	assert.Equal(t, "logstash-env-secret", secrets[0].Name)
	assert.Equal(t, "logstash-file-secret", secrets[1].Name)

	// Test when user specified custom ca certificate
	logstash = &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			LogstashRef: discovercrd.LogstashRef{
				ExternalLogstashRef: &discovercrd.LogstashExternalRef{
					Addresses: []string{
						"logstash1.example.com:9092",
						"logstash2.example.com:9092",
					},
					UserSecretRef: &corev1.LocalObjectReference{
						Name: "external-user-secret",
					},
				},

				LogstashCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-secret",
				},
			},
		},
	}
	secrets, err = buildLogstashSecrets(logstash, nil, nil, secretUserLogstash, secretCustomCa)
	assert.NoError(t, err)
	assert.NotEmpty(t, secrets)
	assert.Len(t, secrets, 2) // Should return both env and file secrets
	assert.Equal(t, secretCustomCa.Data["ca.crt"], secrets[1].Data["ca.crt"])
}

func TestBuildLogstashSecretsManagedLogstash(t *testing.T) {
	expectedLabels := map[string]string{
		"discoverName":                    "test-logstash",
		discovercrd.LogstashAnnotationKey: "true",
		discovercrd.DiscoverAnnotationKey: "true",
	}
	expectedEnvAnnotations := map[string]string{
		discovercrd.LogstashAnnotationKey:                              "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-logstash",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "env",
	}
	expectedFileAnnotations := map[string]string{
		discovercrd.LogstashAnnotationKey:                              "true",
		discovercrd.DiscoverAnnotationKey:                              "true",
		fmt.Sprintf("%s/mountPath", discovercrd.DiscoverAnnotationKey): "test-logstash",
		fmt.Sprintf("%s/type", discovercrd.DiscoverAnnotationKey):      "file",
	}
	logstashCluster := &logstashcrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "managed-logstash-cluster",
			Namespace: "default",
		},
		Spec: logstashcrd.LogstashSpec{
			Services: []shared.Service{
				{
					Name: "beat",
					Spec: corev1.ServiceSpec{
						Type: corev1.ServiceTypeClusterIP,
						Ports: []corev1.ServicePort{
							{
								Name: "beat",
								Port: 80,
							},
						},
					},
				},
			},
		},
	}
	secretUserLogstash := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logstash-user",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"user.key": []byte(`-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCqC7iHvcmK4O7d
hCvKkQ7qfKQSFnSMPhzREfPaRllUI0CpI7MgCUzC6BDjqjTzL+Ai3lfyW+kB0cAE
FJktBh36Le+ObFoP2wTYfSpEIMiD9lwYm6k1Lb2wc8vZlq4zlxkqXtJKs9KNCBCd
Rg/u1iwanxvvibmYCCLvHu+Syi9icUVjc5Owv5Wg+OCh3EI4Wvo9eUb7oE2WMxCh
UukhGf3a6viq1g4uQPh8NrbjB2USIl1rNSINOeg5da7+VZLwE7diRRmoOd6geT6o
sfCOlEKnC9JGXg+B/a0AukwABBBYDNsGpN7pNZpn0zjU9T07i9+14ePeyzGH+7yy
Bvj+WU4JAgMBAAECggEABxCzmxg8noOYCSYrt5/tUbraClJah1UxV1J6olAX+BH6
5swv7h3Uzahcw0jYKF6N3oUEfHJrLQMtCj5c2u0NI87yzCUeXNhJzEDvF6OREUKU
MwWMs0PyRLma97B2Dnixt/v2mtn73pY+MnqtwMuvS2/e9kXvqyxzXyIW7C9wZpOi
54y1SUwrJtce/UMtluStnZtgl8OGO8mDDH14edXojlb9cOy2MdkHEWqmcFI8UJ1I
kVh36cpp+k/TkIAYeE81IME/XYJ5uOvnIfoqVFU2Av9IfYduGQbpc1yJR6Wf34rN
AVbuciaJfA5THC4b5s7OmggqmiadltpcLAafSwRMvQKBgQDY45+PJaQjfNRAtDkW
bo0Qo+qtSLF5hl3CC0fc+hM3mhDM935raP2aoP7kvuZhjsZkQRM4YWQKpPnlBwpn
YHguSSAcAFjBTzNvMagfRH7TGxvjhzFqjxJhsxh6WMjoDF4YPufYGUWHnoIltpPZ
EkHvvIsS1HAdQaQkzeFPOncpfwKBgQDItaP99j2H24nkzfBe1MdgsE8fR47XIpDh
V3mcb2GOGomG/wJam/hZ9cGorwSn3RYakwx0CREFp95r08nPS5JgBIUn5BAxllBt
y7eGyJNhzQ+p2inz+L9ALxQsdBizC2hjD5ItKYjhX5Gli4KSDEQR7Zvzg4TwmXSM
Gmma8eT8dwKBgQCHvgap97v//et1x5YHJQ+iv4RsCZgR3/ekL509f0IxxXoIXkm/
/cjjUynUjmnv2wTt0BCwc7vCMbi5D6wCQU1WYqv2+nySF/iv+hsn3p2lkEznHUDf
WUX+4bFLOXAcU0k80XoXNNHgbGM2KUvlStj7qzw7f7vfg6qw4i3GgWf9DwKBgDjQ
8nNZ2WE0DHzwrzKpGSeNwVghcZYE+u5PTOWzwfHwIS5N0eTEcjFFGGUf5jl+pFsx
dT0DtAXd12C+u0orImx72xXHwa3H10y9dl55OP9ehSiX0Xh22ezZZuSOmM9WU153
cfHG6DCkVR5/t56QSlSo5pSwjlltl0hx0aNDBodRAoGAP2X1Uza3iq0tngKWFIeO
77AxiS6VUb9tjQtQpFF1hmVMMDBLySb15D23M0SzcSzKrtlKxoDSn1FzzETWdSNt
i+x83aUnRlsg3OGRByNVMbj8mJI7m3z/BhN0sUY4ABYE6e/8PLamBnnNe54dhyLs
jymHpXH28vD3JQuliElFEB0=
-----END PRIVATE KEY-----`),
			"user.crt": []byte(`-----BEGIN CERTIFICATE-----
MIIEKzCCAhOgAwIBAgIUVVzGc7Xc5MY1IikS1w08rtP/rywwDQYJKoZIhvcNAQEN
BQAwLTETMBEGA1UECgwKaW8uc3RyaW16aTEWMBQGA1UEAwwNY2xpZW50cy1jYSB2
MDAeFw0yNTA5MTcxNjExNTZaFw0zNTA5MTUxNjExNTZaMBAxDjAMBgNVBAMMBWFk
bWluMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAqgu4h73JiuDu3YQr
ypEO6nykEhZ0jD4c0RHz2kZZVCNAqSOzIAlMwugQ46o08y/gIt5X8lvpAdHABBSZ
LQYd+i3vjmxaD9sE2H0qRCDIg/ZcGJupNS29sHPL2ZauM5cZKl7SSrPSjQgQnUYP
7tYsGp8b74m5mAgi7x7vksovYnFFY3OTsL+VoPjgodxCOFr6PXlG+6BNljMQoVLp
IRn92ur4qtYOLkD4fDa24wdlEiJdazUiDTnoOXWu/lWS8BO3YkUZqDneoHk+qLHw
jpRCpwvSRl4Pgf2tALpMAAQQWAzbBqTe6TWaZ9M41PU9O4vfteHj3ssxh/u8sgb4
/llOCQIDAQABo2AwXjAdBgNVHQ4EFgQUX/lsJNN0QdyvAP7xJYffv80tumQwDAYD
VR0TAQH/BAIwADAOBgNVHQ8BAf8EBAMCBaAwHwYDVR0jBBgwFoAU9xTZqBCn/CmW
aR8tVq7HIGxQpWQwDQYJKoZIhvcNAQENBQADggIBACiDVABQrm4KBlr6AXi1vvIm
9Yjo8sHLukfCCkIr0WIgH4jwKVPBOBYmAhVVml3D2SpnbhVVl0AzUfmk56UTTD5D
aXGeO7d+n9JWzsMC1lmbwce7d1jOhtFAbqFHKr1eF3sTXGkd2TfcJW28slAGk6Ae
2CRi7YWMKs9H6V4xlIe8cMkSmTvB3+RiyDa7LKk/w9vtMShyQr1WtW0AWX+JjuFn
ruWfSny4Nz+sixAqYjEFoddvjJWjRkdsgFWUdvnftIw9qmr8nrOISOnvMEfrtJ5c
xM9Ti53sFvO8bKKubDdJZZHIeSaLNmF1YCo3fb/oKzrpCIseTSCvVqeugn2B/4Mq
G/YtHB6XIQp9qWmBjiJTJ8bkyRz3t87XpKdfr/Y2Ryh1ABwfoSQgnkNsEDyufVIe
UmRSIftNQ1wAWVYN+U74/pqkq0ZBM6pelMEgaGqV7JhNtwY50XuFHoBFGKzIf1WY
AlDRJf5YBbM4SkWGIVXsSHWTkFJq6avkMLdeHEJZH7e5j16tNOI5ROwNALCwMmhz
uu9SSXvmLX73L0Kc8U5tCHd02bL4jBUrC2M0h1yVRfHH7t9a7s38lAsEOCMmEmhR
f4j8bexXdBM/6h6sPlV5nejz2fPb5lKoLS0tCg02n0WC6jNQF/yAdNFM0kzBxspj
S/PdFt4dZmYbx/dqWCV+
-----END CERTIFICATE-----`),
		},
	}
	secretCaLogstash := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "logstash-ca",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte(`-----BEGIN CERTIFICATE-----
MIIFLTCCAxWgAwIBAgIUMA+B6fEMWAfNo6h9erM5ucd9a40wDQYJKoZIhvcNAQEN
BQAwLTETMBEGA1UECgwKaW8uc3RyaW16aTEWMBQGA1UEAwwNY2x1c3Rlci1jYSB2
MDAeFw0yNTA5MTcxNTUxNDdaFw0zNTA5MTUxNTUxNDdaMC0xEzARBgNVBAoMCmlv
LnN0cmltemkxFjAUBgNVBAMMDWNsdXN0ZXItY2EgdjAwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQDgUq+kfwF7fp2QVi30rpAOux9GXyJUQKmSCinW4XIJ
XLo0tjv/YzaWqNQdoXDiY3Q22nsJPE1KHKliNYTaO85VC2/PWZG9CMmfwPcoL7ES
vT1K3zTp7lBR7X3NDcseHyEIx4dCAi/Rs68FTPRjGKRVTfxOBqhuMJIVl+ztn2WJ
E2rnHTl2IpEZaE5fQ9CEfe6mcBEL5O0DnlI6By4r/n71dzkco17GnYDAuebz0ymT
vBs6ByhsZBVC/T8Pqo3eeN99Fj6EQCJnSwgFN1qth02dSrVYyV74HW1XhknjprzH
W7plW3RZOVG6HhLEP7edTnofZf0Dv0tcGtkq1kwNHdGzDW3yIHB0k4imAzYQvLUS
tWWItM1vlyOI92I4kDZZjZSFypfiQJ5HZBmXvJXdwLTMK7T+eDPRoO8UJD90OxPh
0CVcMCe1+BHQe6Qr2+i1pXwKyD+fGh1ggGWDMw0M5GxfK2CDeRTulDn8kgJiE6Fa
bCwkGAw9mon9gocSd6WUR5YK/hELe7H5P4BK+qD2CHt2ogmKE/prlx7tBLqmC4aH
CHUJyAzjMcRp6AMAtewXnyyStLOIVGFFCuHiZRYawx3qkaXMIQYG8zFQamMvR9Aa
EJCWkkaGxSyjatyiSXWMiXO+axm62HEDzF7YGKmVAQxOozqIcH+QcQaQHTtM4ZGE
NQIDAQABo0UwQzAdBgNVHQ4EFgQUvyIeTeFhnS6Od/Mr1ARA6EFx+94wEgYDVR0T
AQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAQYwDQYJKoZIhvcNAQENBQADggIB
AEvcZYOFFMv/t4b7Xsn0AfdBYBPm/qrvqSf30ZJU6dWMt6Iq99+/2w+ljx4JRjXc
xXj4L9l/5Riwo2XlPebabr9uIhLusY1k0aBIyUcClZM17/2A4ndkOjHLD3aRl5ho
b0P/l9xukcQ4giOVp06OG1p0xjy+QnE8Wa/Z2NI0plGyVMeDURl3QhvX/f3sZv0B
fPRNIbBeJCilV7GVNtoJ/zW+hLBX3daXx03Qubjn62DmUiDT6qP6LASdDTHNwTxE
pdVsuLweAfZ7kN65Pu89RIvwQjmlIepE1n73U/9jt8S+zGUS/nOc6I72UWTNRoC2
I1jw0jNefP0+wul39ekuSqzyyCYXRvzS6TFAjMp1ELJ2sCsz1ZOfMZVoP1wqgWH6
OkaKy9BpSGQCuhBAObUE1UPJMCbwwpq4yPCRHVuB0qRJ/fQ6vj+erXfm4CPG9N7k
bXg+WBGBKoTnSjJ1rwGA/8OgIScQOYMRx9txQE0xA7+kI6fb4A6HShX1SJH+1+XT
iYdyQURU58c9nE/CgpUhzmhbEoIC8DI3mu6LYRXBzIQdPSBZxLPG+eWcyebJ4Vfn
LsoolAK9LU9qB9Ad9qjzI6ywAcQt9Gs0KAslsVCapO2Gp/VFjjKrvG+SS+alQF9H
u/w9jG82luQgK3mBxGSVA/H+QYQIspbCrixnUhDlFUdP
-----END CERTIFICATE-----`),
		},
	}
	secretCustomCa := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom-ca-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"ca.crt": []byte(`-----BEGIN CERTIFICATE-----
MIIFLTCCAxWgAwIBAgIULO8C3WB5SH3zMv6hefUvh0lQm6swDQYJKoZIhvcNAQEN
BQAwLTETMBEGA1UECgwKaW8uc3RyaW16aTEWMBQGA1UEAwwNY2xpZW50cy1jYSB2
MDAeFw0yNTA5MTcxNTUxNDhaFw0zNTA5MTUxNTUxNDhaMC0xEzARBgNVBAoMCmlv
LnN0cmltemkxFjAUBgNVBAMMDWNsaWVudHMtY2EgdjAwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQCmk+vJuzR6dlir2LhW0YJagpWPtg4xg2YDPft6yXEi
Ea56YgTz98GmU2WOLQGNlO7AimfO6J7vvH0HEYviZJDgZ3sRC05AvFghHHB9P++3
z1aYfZ9Cgynyo8mnm4sY0+VeQB5beanGOz5kdBx74lPPDPx4sd8SVSCG9GIg7esf
SWsySzxe07FfrlVssAMEuAthDRhvRQ7p8f77zvvGDKswRESuXhqqY7LnuSQK+IF6
/cU3MxRcYhwbaw0dt7aLYWMx75sJTlp/FuKRFepSbQrYfZoFd+mZ9OjostjNM5AW
euC5ldaqQbf7rFtwuDFjEvCTZ4T7JXmM/mLqFL/POIOSqeXtjKrZ5eKWrLOJ7+OP
zQOjtdWhErjpFjRJpiV2OOX8K1v8htqpm4KmOJGmz4Ze6f0cRoC5t9NYkGfhdcRb
B50UtlByGRZ93Kf2/3V1GOnGBSTMQZ/ALHCr+OEVf/gJfo+MandZV1BUArWmQ8ao
XMEBuMJ9ecxpkRzGYM/v059mOdHSbrm8K+T3xNmcUrAqt5XbvJ8TLV9S6eCeMueC
6pyq/HYvFB4UEX/+8hsU5y8Q48Ns0LdPXXhr/ZNPjY72NgAhAdLlqW2XTNDJ29g6
jCDkTUwem9trNulNePVrf4QASEqjDR/SiIrrjQzN1aKe25lF8Pvxw9wuz6rEAnq8
UwIDAQABo0UwQzAdBgNVHQ4EFgQU9xTZqBCn/CmWaR8tVq7HIGxQpWQwEgYDVR0T
AQH/BAgwBgEB/wIBADAOBgNVHQ8BAf8EBAMCAQYwDQYJKoZIhvcNAQENBQADggIB
ABBJLRijdoszHAqaaScN8Vrbq1uyzo3f0xKXIMlhZi1U0g8G4A9G0Epwj1/d5cBk
iPfEhQJOvheVSjtGokvc9eHUA+I57lZLNfXn1zxpArfJ9nUw3AgKgHRosDayQI7c
8wACXniqbJiklP7RvpHIU04BST7K5BWu3YPXbEPx6cgJpDG0wvE7+AFuTsWKcGMF
S1mNVVfL3Xk6H3oo/JOv04Pm3p3Mv8LoQvCihvHBBTN785IWibgZPMfz7tomKWoF
oZupvonxIaRPBrECAxIw56Yfcp5RWaKa13rC1hMScsWHRKpnt2T6nRqUnyQTF2aN
Va8wDl8au42GXAxMqQKAnJ/7nLKHalJMjLCk66cn2e564R3qvsrD40MaN1tZgSnx
UdZUeP5AulqI7C3G2/wC8Mf/myaHGL0V/zJMNUjuWC7r6QEjIuSl2Qgb81PzIuVB
ivVZ9IxC54E1uOia36kOb59NV1AdlkbPuCQMRSaLXVXxapP5atVyL5bleUJAP57U
fkGjZreCovXaLcoG2zXH8+l1r3Gb+uVOroKOXMewcYE1eqEcJGjn8KtI8N7x6Bca
HLjcU22GslLop0zHhqTSDlXkoGpzM4WfMj8i8KwboFXUfAGQYePJWGqM7kIbw2yh
pfJ8FlBrI6VNf5xHO14hLGS2A+WHopI8rosEUjssToEC
-----END CERTIFICATE-----`),
		},
	}

	// Test with managed Logstash configuration without discovered target secrets
	logstash := &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			LogstashRef: discovercrd.LogstashRef{
				ManagedLogstashRef: &discovercrd.LogstashManagedRef{
					Name: "managed-logstash-cluster",
					UserRef: &corev1.SecretReference{
						Name: "test",
					},
					TargetService: "beat",
					Port:          80,
				},
			},
		},
	}
	secrets, err := buildLogstashSecrets(logstash, logstashCluster, nil, secretUserLogstash, nil)

	assert.NoError(t, err)
	assert.NotNil(t, secrets)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "test-logstash-env", secrets[0].Name)
	assert.Equal(t, "default", secrets[0].Namespace)
	assert.Equal(t, expectedLabels, secrets[0].Labels)
	assert.Equal(t, expectedEnvAnnotations, secrets[0].Annotations)
	assert.Equal(t, "managed-logstash-cluster-beat-ls.default.svc:80", string(secrets[0].Data["LOGSTASH_HOSTS_TEST_LOGSTASH"]))
	assert.NotEmpty(t, secrets[0].Data["LOGSTASH_USER_PASSWORD_TEST_LOGSTASH"])
	assert.Equal(t, "test-logstash-file", secrets[1].Name)
	assert.Equal(t, "default", secrets[1].Namespace)
	assert.Equal(t, expectedLabels, secrets[1].Labels)
	assert.Equal(t, expectedFileAnnotations, secrets[1].Annotations)
	assert.Equal(t, string(secretUserLogstash.Data["user.crt"]), string(secrets[1].Data["user.crt"]))
	assert.Equal(t, string(secretUserLogstash.Data["user.key"]), string(secrets[1].Data["user.key"]))
	assert.NotEmpty(t, secrets[1].Data["user.p12"])

	// Test when user specified target secrets
	logstash = &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "logstash-env-secret",
				},
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "logstash-file-secret",
				},
			},
			LogstashRef: discovercrd.LogstashRef{
				ManagedLogstashRef: &discovercrd.LogstashManagedRef{
					Name: "managed-logstash-cluster",
					UserRef: &corev1.SecretReference{
						Name: "logstash-user",
					},
				},
			},
		},
	}
	secrets, err = buildLogstashSecrets(logstash, logstashCluster, nil, secretUserLogstash, nil)

	assert.NoError(t, err)
	assert.NotNil(t, secrets)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "logstash-env-secret", secrets[0].Name)
	assert.Equal(t, "logstash-file-secret", secrets[1].Name)

	// Test when internal PKI is enable on Logstash cluster
	logstash = &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "logstash-env-secret",
				},
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "logstash-file-secret",
				},
			},
			LogstashRef: discovercrd.LogstashRef{
				ManagedLogstashRef: &discovercrd.LogstashManagedRef{
					Name: "managed-logstash-cluster",
				},
			},
		},
	}
	secrets, err = buildLogstashSecrets(logstash, logstashCluster, secretCaLogstash, secretUserLogstash, nil)

	assert.NoError(t, err)
	assert.NotNil(t, secrets)
	assert.Len(t, secrets, 2)
	assert.NotEmpty(t, secrets[1].Data["ca.crt"])
	assert.Equal(t, string(secretCaLogstash.Data["ca.crt"]), string(secrets[1].Data["ca.crt"]))

	// Test when user specified extra custom CA certificate
	logstash = &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			Discover: discovercrd.Discover{
				TargetSecretEnvRef: &corev1.LocalObjectReference{
					Name: "logstash-env-secret",
				},
				TargetSecretFileRef: &corev1.LocalObjectReference{
					Name: "logstash-file-secret",
				},
			},
			LogstashRef: discovercrd.LogstashRef{
				ManagedLogstashRef: &discovercrd.LogstashManagedRef{
					Name: "managed-logstash-cluster",
				},
				LogstashCaSecretRef: &corev1.LocalObjectReference{
					Name: "custom-ca-secret",
				},
			},
		},
	}
	secrets, err = buildLogstashSecrets(logstash, logstashCluster, secretCaLogstash, secretUserLogstash, secretCustomCa)

	assert.NoError(t, err)
	assert.NotNil(t, secrets)
	assert.Len(t, secrets, 2)
	assert.NotEmpty(t, secrets[1].Data["ca.crt"])
	assert.Equal(t, string(secretCaLogstash.Data["ca.crt"])+"\n"+string(secretCustomCa.Data["ca.crt"]), string(secrets[1].Data["ca.crt"]))

	// Test when set env suffix
	logstash = &discovercrd.Logstash{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-logstash",
			Namespace: "default",
		},
		Spec: discovercrd.LogstashSpec{
			LogstashRef: discovercrd.LogstashRef{
				ManagedLogstashRef: &discovercrd.LogstashManagedRef{
					Name:          "managed-logstash-cluster",
					TargetService: "beat",
					Port:          80,
				},
			},

			Discover: discovercrd.Discover{
				Name: ptr.To("CUSTOM"),
			},
		},
	}
	secrets, err = buildLogstashSecrets(logstash, logstashCluster, nil, secretUserLogstash, nil)

	assert.NoError(t, err)
	assert.NotNil(t, secrets)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "managed-logstash-cluster-beat-ls.default.svc:80", string(secrets[0].Data["LOGSTASH_HOSTS_CUSTOM"]))
	assert.NotEmpty(t, secrets[0].Data["LOGSTASH_USER_PASSWORD_CUSTOM"])
}
