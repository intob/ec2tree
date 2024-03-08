package vm

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type Cfg struct {
	AMI          string
	InstanceType string
	MaxDepth     int
	Fanout       int
	Region       string
}

type Svc struct {
	ec2 *ec2.EC2
	cfg *Cfg
}

type Node struct {
	Instance *ec2.Instance
	Children []*Node
	KeyPair  *ec2.CreateKeyPairOutput
}

func NewSvc(cfg *Cfg) (*Svc, error) {
	// Create AWS session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.Region),
	})
	if err != nil {
		return nil, err
	}

	// Create EC2 service client
	return &Svc{
		ec2: ec2.New(sess),
		cfg: cfg,
	}, nil
}

func (s *Svc) CreateTree() (*Node, error) {
	rootKeyPair, err := s.ec2.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String("root"),
	})
	if err != nil {
		return nil, fmt.Errorf("err creating root key pair: %w", err)
	}

	rootInstance, err := s.createInstance(*rootKeyPair.KeyName)
	if err != nil {
		return nil, fmt.Errorf("err creating root instance: %v", err)
	}

	fmt.Printf("Root instance ID: %s\n", *rootInstance.InstanceId)
	rootNode := &Node{
		Instance: rootInstance,
		KeyPair:  rootKeyPair,
	}

	return rootNode, s.createTree(rootNode, 0)
}

func (s *Svc) DeleteTree(n *Node) error {
	// First, terminate the instance
	terminateInput := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{n.Instance.InstanceId},
	}
	_, err := s.ec2.TerminateInstances(terminateInput)
	if err != nil {
		return fmt.Errorf("failed to terminate instance %s: %v", *n.Instance.InstanceId, err)
	}

	// Delete the key pair
	deleteKeyInput := &ec2.DeleteKeyPairInput{
		KeyName: n.KeyPair.KeyName,
	}
	_, err = s.ec2.DeleteKeyPair(deleteKeyInput)
	if err != nil {
		return fmt.Errorf("failed to delete key pair %s: %v", *n.KeyPair.KeyName, err)
	}

	// Recursively terminate & delete children
	for _, child := range n.Children {
		err = s.DeleteTree(child)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Svc) createTree(root *Node, depth int) error {
	if depth >= s.cfg.MaxDepth {
		return nil
	}

	root.Children = make([]*Node, s.cfg.Fanout)
	for i := 0; i < s.cfg.Fanout; i++ {
		child, err := s.createChildNode(root, depth, i)
		if err != nil {
			return fmt.Errorf("err creating node at depth %d, child %d: %w", depth, i, err)
		}
		root.Children[i] = child
		err = s.createTree(child, depth+1)
		if err != nil {
			return fmt.Errorf("err creating tree at depth %d, child %d: %w", depth, i, err)
		}
	}

	return nil
}

func (s *Svc) createChildNode(parent *Node, depth, childIndex int) (*Node, error) {
	keyPair, err := s.ec2.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String(fmt.Sprintf("child-%d-%d", depth, childIndex)),
	})
	if err != nil {
		return nil, err
	}

	instance, err := s.createInstance(*keyPair.KeyName)
	if err != nil {
		return nil, fmt.Errorf("err creating instance at depth %d, child %d: %w", depth, childIndex, err)
	}

	child := &Node{
		Instance: instance,
		KeyPair:  keyPair,
	}

	fmt.Printf("%s instance ID: %s\n", *child.KeyPair.KeyName, *child.Instance.InstanceId)
	//appendHostsFile(parent, child)

	return child, nil
}

func (s *Svc) createInstance(keyName string) (*ec2.Instance, error) {
	resp, err := s.ec2.RunInstances(&ec2.RunInstancesInput{
		ImageId:      aws.String(s.cfg.AMI),
		InstanceType: aws.String(s.cfg.InstanceType),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		KeyName:      aws.String(keyName),
	})
	if err != nil {
		return nil, err
	}

	return resp.Instances[0], nil
}

/*
func (s *Svc) writePrivateKeyToParent(parent *Node, privateKey *string) error {
	return nil
}

func appendHostsFile(parent, child *Node) error {
	// Implement logic to append the child's IP address to the parent's hosts file
	// This will depend on how you manage the hosts file on the parent instance
	// (e.g., using a user data script, SSH, or other methods)
	return nil
}
*/
