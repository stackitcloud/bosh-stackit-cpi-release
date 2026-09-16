package helpers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stackitcloud/stackit-cpi/cpi/lib"
	"github.com/stackitcloud/stackit-sdk-go/core/utils"
	iaas "github.com/stackitcloud/stackit-sdk-go/services/iaas/v2api"
	"golang.org/x/crypto/ssh"
)

const (
	TestSSHKeyName   = "test-ssh-key"
	DefaultKeyLength = 2048
)

type TestSSHKey struct {
	Name   string
	PKPath string
	client iaas.APIClient
}

func (t TestSSHKey) Delete() {
	Expect(t.client.DefaultAPI.DeleteKeyPair(context.Background(), t.Name).Execute()).To(Succeed())
}

func (p TestProject) SSHTunnelConnection(target string, privateKeyPath string) *ssh.Client {
	// create Jumpbox ssh config
	pkBytes := []byte(p.Jumpbox.PrivateKeyPemBlock)
	signer, err := ssh.ParsePrivateKey(pkBytes)
	Expect(err).ToNot(HaveOccurred())

	auth := ssh.PublicKeys(signer)
	config := &ssh.ClientConfig{
		User:            "ubuntu",
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	// start dialing. all this should be started from scratch if any of the stages fail.
	// we're shoving connections through connections here and if one of the lower connections
	// fail ( e.g. the tunnel to the jumphost, everything should be rebuild from scratch.

	// if the VM was just created it can take a while for the ssh server to come up... give it a few minutes
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// incremental backoff: 1s, 2s, 4s, 8s, 10s (cap)
	// protects the target SSH daemon from being flooded with connection attempts
	var delay int64 = 1000       // ms
	const maxDelay int64 = 10000 // ms

	attempt := 1
	for ctx.Err() == nil {
		fmt.Println("starting", "attempt", attempt)
		tunnelClient, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", p.Jumpbox.PublicIP), config)
		if err != nil {
			fmt.Println("failed creating tunnelClient", "target", fmt.Sprintf("%s:22", p.Jumpbox.PublicIP), "err", err.Error())
			attempt += 1
			lib.JitterWait(delay)
			delay = min(delay*2, maxDelay)
			continue
		}
		fmt.Println(p.Jumpbox.PublicIP, "tunnel session established")

		dialCtx, dialCancel := context.WithTimeout(context.Background(), 15*time.Second)

		targetNetConnection, dialErr := tunnelClient.DialContext(dialCtx, "tcp", fmt.Sprintf("%s:22", target))
		dialCancel()
		if dialErr != nil {
			fmt.Printf("error creating targetNetConnection: %s\n", dialErr)
			tunnelClient.Close()
			attempt += 1
			lib.JitterWait(delay)
			delay = min(delay*2, maxDelay)
			continue
		}

		fmt.Println("targetNetConnection established")

		targetPK, err := os.ReadFile(privateKeyPath)
		Expect(err).ToNot(HaveOccurred())
		signer, err = ssh.ParsePrivateKey(targetPK)
		Expect(err).ToNot(HaveOccurred())
		auth = ssh.PublicKeys(signer)

		targetConfig := &ssh.ClientConfig{
			User:            "vcap",
			Auth:            []ssh.AuthMethod{auth},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         time.Minute,
		}
		targetSSHConnection, newChannelChannel, requestChannel, handshakeErr := ssh.NewClientConn(targetNetConnection, target, targetConfig)
		if handshakeErr != nil {
			fmt.Println("failed creating targetSSHConnection", "error", handshakeErr)
			attempt += 1
			// ensure everything opened so far is closed
			tunnelClient.Close()
			if targetNetConnection != nil {
				targetNetConnection.Close()
			}
			lib.JitterWait(int64(delay))
			delay = min(delay*2, maxDelay)
			continue
		}

		return ssh.NewClient(targetSSHConnection, newChannelChannel, requestChannel)
	}
	// we shouldn't ever be here. so if we are, ensure we're failing
	Expect(true).To(BeFalse())
	return nil
}

func (p TestProject) GenerateSSHKey() TestSSHKey {
	client := p.GetStackitIaasClient()
	privateKey, err := rsa.GenerateKey(rand.Reader, DefaultKeyLength)
	Expect(err).ToNot(HaveOccurred())
	// generate and write private key as PEM
	privateKeyFile, err := os.CreateTemp("", "")
	Expect(err).ToNot(HaveOccurred())
	defer privateKeyFile.Close()

	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	Expect(pem.Encode(privateKeyFile, privateKeyPEM)).To(Succeed())

	// generate and write public key
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)

	Expect(err).ToNot(HaveOccurred())

	keyPairPayload := client.DefaultAPI.CreateKeyPair(context.Background())
	keyPair, err := keyPairPayload.CreateKeyPairPayload(iaas.CreateKeyPairPayload{
		Name:      utils.Ptr(filepath.Base(privateKeyFile.Name())),
		PublicKey: string(ssh.MarshalAuthorizedKey(pub)),
	}).Execute()
	Expect(err).ToNot(HaveOccurred())

	return TestSSHKey{
		Name:   keyPair.GetName(),
		PKPath: privateKeyFile.Name(),
		client: *client,
	}
}
