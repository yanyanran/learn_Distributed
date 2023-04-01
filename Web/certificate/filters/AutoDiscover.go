package filters

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
)

func AutoDiscover() {
	// 获取Vault客户端对象
	vaultClient, err := getVaultClient()
	if err != nil {
		fmt.Println("Failed to create Vault client:", err)
		os.Exit(1)
	}

	// 从Vault获取TLS证书和秘钥
	cert, err := getVaultCert(vaultClient, "path/to/cert")
	if err != nil {
		log.Fatal(err)
	}

	key, err := getVaultKey(vaultClient, "path/to/key")
	if err != nil {
		log.Fatal(err)
	}

	// 从Vault获取CA证书
	caCert, err := getVaultCACert(vaultClient, "path/to/ca")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Certificate:", cert)
	fmt.Println("Private key:", key)

	// 使用证书和TLS配置创建一个TLS客户端
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      x509.NewCertPool(),
	}
	tlsConfig.RootCAs.AppendCertsFromPEM(caCert)

	// 建立与服务器的安全连接
	// 在这里使用证书和密钥进行TLS连接
	conn, err := tls.Dial("tcp", "example.com:443", tlsConfig)
	if err != nil {
		log.Fatal(err)
	}

	// 使用连接安全通信
	conn.Write([]byte("hello, world\n"))
	conn.Close()
}

func getVaultClient() (*api.Client, error) {
	// 创建Vault客户端配置
	vaultConfig := api.DefaultConfig()
	vaultConfig.Address = "http://127.0.0.1:8200"

	// 设置Vault Token
	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultToken == "" {
		return nil, fmt.Errorf("VAULT_TOKEN environment variable not set")
	}
	vaultConfig.SetToken(vaultToken)

	// 创建Vault客户端
	//client, err := api.NewClient(api.DefaultConfig())
	vaultClient, err := api.NewClient(vaultConfig)
	if err != nil {
		return nil, err
	}

	return vaultClient, nil
}

func getVaultCert(client *api.Client, certPath string) (tls.Certificate, error) {
	secret, err := client.Logical().Read(certPath) // 使用Vault API读取证书
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM, ok := secret.Data["certificate"].(string)
	if !ok {
		return tls.Certificate{}, fmt.Errorf("invalid certificate format")
	}

	keyPEM, ok := secret.Data["private_key"].(string)
	if !ok {
		return tls.Certificate{}, fmt.Errorf("invalid private key format")
	}

	cert, err := tls.X509KeyPair([]byte(certPEM), []byte(keyPEM))
	if err != nil {
		return tls.Certificate{}, err
	}

	return cert, nil
}

func getVaultKey(client *api.Client, keyPath string) ([]byte, error) {
	secret, err := client.Logical().Read(keyPath)
	if err != nil {
		return nil, err
	}

	key, ok := secret.Data["private_key"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid private key format")
	}

	return []byte(key), nil
}

func getVaultCACert(client *api.Client, caPath string) ([]byte, error) {
	secret, err := client.Logical().Read(caPath)
	if err != nil {
		return nil, err
	}

	caCert, ok := secret.Data["certificate"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid CA certificate format")
	}

	return []byte(caCert), nil
}
