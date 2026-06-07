package test

import "context"

type FakePodPort struct {
	AddLabelCalled    bool
	AddLabelServer    string
	AddLabelNamespace string
	AddLabelKey       string
	AddLabelValue     string
	AddLabelErr       error

	RemoveLabelCalled    bool
	RemoveLabelServer    string
	RemoveLabelNamespace string
	RemoveLabelKey       string
	RemoveLabelErr       error
}

func (f *FakePodPort) AddPodLabel(ctx context.Context, serverName, namespace, key, value string) error {
	f.AddLabelCalled = true
	f.AddLabelServer = serverName
	f.AddLabelNamespace = namespace
	f.AddLabelKey = key
	f.AddLabelValue = value
	return f.AddLabelErr
}

func (f *FakePodPort) RemovePodLabel(ctx context.Context, serverName, namespace, key string) error {
	f.RemoveLabelCalled = true
	f.RemoveLabelServer = serverName
	f.RemoveLabelNamespace = namespace
	f.RemoveLabelKey = key
	return f.RemoveLabelErr
}
