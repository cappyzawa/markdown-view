package markdownview

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"

	viewv1 "github.com/cappyzawa/markdown-view/api/v1"
)


func mutateTest(before string, after string) {
	ctx := context.Background()

	y, err := os.ReadFile(before)
	Expect(err).To(Succeed())
	d := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(y), 4096)
	beforeView := &viewv1.MarkdownView{}
	err = d.Decode(beforeView)
	Expect(err).To(Succeed())

	err = k8sClient.Create(ctx, beforeView)
	Expect(err).To(Succeed())

	ret := &viewv1.MarkdownView{}
	err = k8sClient.Get(ctx, types.NamespacedName{Name: beforeView.GetName(), Namespace: beforeView.GetNamespace()}, ret)
	Expect(err).To(Succeed())

	y, err = os.ReadFile(after)
	Expect(err).To(Succeed())
	d = yaml.NewYAMLOrJSONDecoder(bytes.NewReader(y), 4096)
	afterView := &viewv1.MarkdownView{}
	err = d.Decode(afterView)
	Expect(err).To(Succeed())

	Expect(ret.Spec).Should(Equal(afterView.Spec))
}

func validateTest(file string, valid bool) {
	ctx := context.Background()
	y, err := os.ReadFile(file)
	Expect(err).To(Succeed())
	d := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(y), 4096)
	mdView := &viewv1.MarkdownView{}
	err = d.Decode(mdView)
	Expect(err).To(Succeed())

	err = k8sClient.Create(ctx, mdView)

	if valid {
		Expect(err).To(Succeed(), "MarkdownView: %v", mdView)
	} else {
		Expect(err).To(HaveOccurred(), "MarkdownView: %v", mdView)
		statusErr := &apierrors.StatusError{}
		Expect(errors.As(err, &statusErr)).To(BeTrue())
		expected := mdView.Annotations["message"]
		Expect(statusErr.ErrStatus.Message).To(ContainSubstring(expected))
	}
}

var _ = Describe("MarkdownView Handler", func() {
	Context("when mutating", func() {
		It("should mutate a MarkdownView", func() {
			mutateTest(filepath.Join("testdata", "mutating", "before.yaml"), filepath.Join("testdata", "mutating", "after.yaml"))
		})
	})
	Context("when validating", func() {
		It("should create a valid MarkdownView", func() {
			validateTest(filepath.Join("testdata", "validating", "valid.yaml"), true)
			validateTest(filepath.Join("testdata", "validating", "non-default-img.yaml"), true)
		})
		It("should not create an invalid MarkdownView", func() {
			validateTest(filepath.Join("testdata", "validating", "empty-markdowns.yaml"), false)
			validateTest(filepath.Join("testdata", "validating", "invalid-replicas.yaml"), false)
			validateTest(filepath.Join("testdata", "validating", "without-summary.yaml"), false)
		})
	})
})
