package test

import (
	"sync/atomic"

	ocfCloud "github.com/plgd-dev/device/v2/pkg/ocf/cloud"
)

type ServiceHandler = interface {
	CloseOnError() bool
	SignUp(req ocfCloud.CoapSignUpRequest) (ocfCloud.CoapSignUpResponse, error)
	SignOff() error
	SignIn(req ocfCloud.CoapSignInRequest) (ocfCloud.CoapSignInResponse, error)
	SignOut(req ocfCloud.CoapSignInRequest) error
	PublishResources(req ocfCloud.PublishResourcesRequest) error
	UnpublishResources(req ocfCloud.UnpublishResourcesRequest) error
	RefreshToken(req ocfCloud.CoapRefreshTokenRequest) (ocfCloud.CoapRefreshTokenResponse, error)
}

type (
	RefreshTokenFunc       func(req ocfCloud.CoapRefreshTokenRequest) (ocfCloud.CoapRefreshTokenResponse, error)
	SignInFunc             func(req ocfCloud.CoapSignInRequest) (ocfCloud.CoapSignInResponse, error)
	SignOutFunc            func(req ocfCloud.CoapSignInRequest) error
	SignOffFunc            func() error
	SignUpFunc             func(req ocfCloud.CoapSignUpRequest) (ocfCloud.CoapSignUpResponse, error)
	PublishResourcesFunc   func(req ocfCloud.PublishResourcesRequest) error
	UnpublishResourcesFunc func(req ocfCloud.UnpublishResourcesRequest) error
)

type CustomHandler struct {
	h                  ServiceHandler
	refreshToken       atomic.Pointer[RefreshTokenFunc]
	signIn             atomic.Pointer[SignInFunc]
	signOut            atomic.Pointer[SignOutFunc]
	signOff            atomic.Pointer[SignOffFunc]
	signUp             atomic.Pointer[SignUpFunc]
	publishResources   atomic.Pointer[PublishResourcesFunc]
	unpublishResources atomic.Pointer[UnpublishResourcesFunc]
}

func NewCustomHandler(s ServiceHandler) *CustomHandler {
	return &CustomHandler{
		h: s,
	}
}

func (h *CustomHandler) SetRefreshToken(f RefreshTokenFunc) {
	h.refreshToken.Store(&f)
}

func (h *CustomHandler) SetSignIn(f SignInFunc) {
	h.signIn.Store(&f)
}

func (h *CustomHandler) SetSignOut(f SignOutFunc) {
	h.signOut.Store(&f)
}

func (h *CustomHandler) SetSignOff(f SignOffFunc) {
	h.signOff.Store(&f)
}

func (h *CustomHandler) SetSignUp(f SignUpFunc) {
	h.signUp.Store(&f)
}

func (h *CustomHandler) SetPublishResources(f PublishResourcesFunc) {
	h.publishResources.Store(&f)
}

func (h *CustomHandler) SetUnpublishResources(f UnpublishResourcesFunc) {
	h.unpublishResources.Store(&f)
}

func (h *CustomHandler) SignUp(req ocfCloud.CoapSignUpRequest) (ocfCloud.CoapSignUpResponse, error) {
	f := h.signUp.Load()
	if f == nil {
		return h.h.SignUp(req)
	}
	return (*f)(req)
}

func (h *CustomHandler) CloseOnError() bool {
	return h.h.CloseOnError()
}

func (h *CustomHandler) SignOff() error {
	f := h.signOff.Load()
	if f == nil {
		return h.h.SignOff()
	}
	return (*f)()
}

func (h *CustomHandler) SignIn(req ocfCloud.CoapSignInRequest) (ocfCloud.CoapSignInResponse, error) {
	f := h.signIn.Load()
	if f == nil {
		return h.h.SignIn(req)
	}
	return (*f)(req)
}

func (h *CustomHandler) SignOut(req ocfCloud.CoapSignInRequest) error {
	f := h.signOut.Load()
	if f == nil {
		return h.h.SignOut(req)
	}
	return (*f)(req)
}

func (h *CustomHandler) PublishResources(req ocfCloud.PublishResourcesRequest) error {
	f := h.publishResources.Load()
	if f == nil {
		return h.h.PublishResources(req)
	}
	return (*f)(req)
}

func (h *CustomHandler) UnpublishResources(req ocfCloud.UnpublishResourcesRequest) error {
	f := h.unpublishResources.Load()
	if f == nil {
		return h.h.UnpublishResources(req)
	}
	return (*f)(req)
}

func (h *CustomHandler) RefreshToken(req ocfCloud.CoapRefreshTokenRequest) (ocfCloud.CoapRefreshTokenResponse, error) {
	f := h.refreshToken.Load()
	if f == nil {
		return h.h.RefreshToken(req)
	}
	return (*f)(req)
}
