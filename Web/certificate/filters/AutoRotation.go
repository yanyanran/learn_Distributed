package filters

import (
	"crypto/tls"
	"github.com/hashicorp/vault/api"
	"time"
)

func autoRotateCert(cert *tls.Certificate, rotationInterval time.Duration) error {
	for {
		// 等待一段时间
		time.Sleep(rotationInterval)

		// 轮换证书
		err := rotateCert(cert)
		if err != nil {
			return err
		}
	}
}

func rotateCert(cert *tls.Certificate) error {
	// 获取新的证书和私钥
	newCert, err := getNewCertFromVault()
	if err != nil {
		return err
	}
	// 更新证书和私钥
	cert.Certificate = append(cert.Certificate[:0], newCert.Certificate...)
	cert.PrivateKey = newCert.PrivateKey

	return nil
}

func getNewCertFromVault() (*tls.Certificate, error) {
	// 创建Vault客户端
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = "http://127.0.0.1:8200"
	vaultClient, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}

	// 获取新的证书和私钥
	secret, err := vaultClient.Logical().Read("path/to/new/cert")
	if err != nil {
		return nil, err
	}
	// 解析证书和私钥
	certPEM := secret.Data["cert"].(string)
	keyPEM := secret.Data["key"].(string)
	newCert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return nil, err
	}

	return &newCert, nil
}
