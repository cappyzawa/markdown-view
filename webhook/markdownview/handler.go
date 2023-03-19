package markdownview

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	viewv1 "github.com/cappyzawa/markdown-view/api/v1"
	admissionv1 "k8s.io/api/admission/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	_      admission.Handler = &Mutator{}
	_      admission.Handler = &Validator{}
	logger                   = logf.Log.WithName("markdownview.webhook")
)

type Mutator struct {
	decoder *admission.Decoder
}

func (m *Mutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger.Info("mutation", "name", req.Name, "namespace", req.Namespace)

	var mdView viewv1.MarkdownView
	if err := m.decoder.Decode(req, &mdView); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if len(mdView.Spec.ViewerImage) == 0 {
		mdView.Spec.ViewerImage = "peaceiris/mdbook:latest"
	}
	marshaled, err := json.Marshal(mdView)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaled)
}

func (m *Mutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

type Validator struct {
	decoder *admission.Decoder
}

func (v *Validator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger.Info("validating", "name", req.Name, "namespace", req.Namespace)

	var mdView viewv1.MarkdownView
	if err := v.decoder.Decode(req, &mdView); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case admissionv1.Create, admissionv1.Update:
		if err := v.validateError(ctx, &mdView); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		if msgs := v.validateWarn(ctx, &mdView); len(msgs) != 0 {
			return admission.Allowed("").WithWarnings(msgs...)
		}
		return admission.Allowed("")
	default:
		return admission.Allowed("")
	}
}

func (v *Validator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

func (v *Validator) validateError(ctx context.Context, mdView *viewv1.MarkdownView) error {
	logger := logf.FromContext(ctx)
	var errs field.ErrorList
	if mdView.Spec.Replicas < 1 || mdView.Spec.Replicas > 5 {
		errs = append(errs, field.Invalid(field.NewPath("spec", "replicas"), mdView.Spec.Replicas, "replicas must be in the range of 1 to 5."))
	}
	hasSummary := false
	for name := range mdView.Spec.Markdowns {
		if name == "SUMMARY.md" {
			hasSummary = true
		}
	}
	if !hasSummary {
		errs = append(errs, field.Required(field.NewPath("spec", "markdowns"), "markdowns must have SUMMARY.md."))
	}

	if len(errs) > 0 {
		err := apierrors.NewInvalid(schema.GroupKind{Group: viewv1.GroupVersion.Group, Kind: "MarkdownView"}, mdView.Name, errs)
		logger.Error(err, "validation error", "name", mdView.Name)
		return err
	}

	return nil
}

func (v *Validator) validateWarn(ctx context.Context, mdView *viewv1.MarkdownView) []string {
	var warnMsgs []string
	isDefaultViewerImage := false
	if mdView.Spec.ViewerImage == "peaceiris/mdbook:latest" {
		isDefaultViewerImage = true
	}
	if !isDefaultViewerImage {
		warnMsgs = append(warnMsgs, fmt.Sprintf("%s: \"%s\": viewerImage is not default", field.NewPath("spec", "viewerImage"), mdView.Spec.ViewerImage))
	}

	return warnMsgs
}
