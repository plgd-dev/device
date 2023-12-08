package main

/*
#include <stdio.h>
#include <pthread.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>

typedef void (*ResponseCallback) (void *data, size_t len, void* user_data);

struct thread_data {
	ResponseCallback callback;
	void *data;
	size_t len;
	void* user_data;
};

static void free_thread_data(struct thread_data *thrd) {
	if (!thrd) {
		return;
	}
	free(thrd->data);
	free(thrd);
}

static inline void* threadWrapperCallback(void *rawthrd) {
	struct thread_data *thrd = (struct thread_data *) rawthrd;
	thrd->callback(thrd->data, thrd->len, thrd->user_data);
	free_thread_data(thrd);
	return NULL;
}

static inline void bridgeResponseCallback(ResponseCallback callback, void *data, size_t len, void *user_data)
{
	pthread_t thread1;
	pthread_attr_t attr;

	struct thread_data *thrd = (struct thread_data *)malloc(sizeof(*thrd));
	thrd->data = data;
	thrd->len = len;
	thrd->callback = callback;
	thrd->user_data = user_data;

	int rc = pthread_attr_init(&attr);
	if (rc != 0) {
		fprintf(stderr, "error: bridgeResponseCallback: pthread_attr_init returns non-zero value(%d)\n", rc);
		free_thread_data(thrd);
		return;
	}

	rc = pthread_attr_setdetachstate(&attr, PTHREAD_CREATE_DETACHED);
	if (rc != 0) {
		fprintf(stderr, "error: bridgeResponseCallback: pthread_attr_setdetachstate returns non-zero value(%d)\n", rc);
		free_thread_data(thrd);
		pthread_attr_destroy(&attr);
		return;
	}

	rc = pthread_create(&thread1, &attr, threadWrapperCallback, thrd);
	if (rc != 0) {
		fprintf(stderr, "error: bridgeResponseCallback: pthread_create returns non-zero value(%d)\n", rc);
		free_thread_data(thrd);
	}
	pthread_attr_destroy(&attr);
}

// Error codes that is returned by C license functions.
enum {
    kiconnect_sdk_OK                  =  0, // ok
    kiconnect_sdk_ERROR               = -1, // generic error
    kiconnect_sdk_INVALID_ARG         = -2, // invalid argument in function
	kiconnect_sdk_INSUFFICIENT_BUFFER = -3, // insufficient buffer is in argument, read 'resp_len' argument for resize buffer
	kiconnect_sdk_NOT_SUPPORTED       = -4, // function is not supported
};


typedef void* kiconnect_sdk_handler_t;
typedef int(*kiconnect_sdk_application_callback)(void* req, size_t len, void* resp, size_t resp_size, size_t* resp_len, void *user_data);

struct app_thread_data {
	kiconnect_sdk_application_callback callback;
	void* req;
	size_t len;
	void* resp;
	size_t resp_size;
	size_t* resp_len;
	void *user_data;
	int ret;
};

static inline void* threadWrapperAppCallback(void *rawthrd) {
	struct app_thread_data *thrd = (struct app_thread_data *) rawthrd;
	thrd->ret = thrd->callback(thrd->req, thrd->len, thrd->resp, thrd->resp_size, thrd->resp_len, thrd->user_data);
	return NULL;
}

static inline int bridgeApplicationCallback(kiconnect_sdk_application_callback callback, void* req, size_t len, void* resp, size_t resp_size, size_t* resp_len, void* user_data)
{
	pthread_t thread1;

	struct app_thread_data *thrd = (struct app_thread_data *)malloc(sizeof(*thrd));
	thrd->callback = callback;
	thrd->user_data = user_data;
	thrd->req = req;
	thrd->len = len;
	thrd->resp = resp;
	thrd->resp_size = resp_size;
	thrd->resp_len = resp_len;
	thrd->ret = 0;

	int rc = pthread_create(&thread1, NULL, threadWrapperAppCallback, thrd);
	if (rc != 0) {
		fprintf(stderr, "error: bridgeApplicationCallback: pthread_create returns non-zero value(%d)\n", rc);
		free(thrd);
		return kiconnect_sdk_ERROR;
	}
	rc = pthread_join(thread1, NULL);
	if (rc != 0) {
		fprintf(stderr, "error: bridgeApplicationCallback: pthread_join returns non-zero value(%d)\n", rc);
		free(thrd);
		return kiconnect_sdk_ERROR;
	}
	rc = thrd->ret;
	free(thrd);
	return rc;
}

*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

type NativeClient struct {
}

type AppCallback = interface {
	CallApplication(request []byte) ([]byte, error)
}

func NewClient(data []byte, appCallback AppCallback) (*NativeClient, error) {
	return nil, nil
}

// BinaryCallback accepts a marshalled response.
type BinaryCallback interface {
	Respond(data []byte)
}

func (c *NativeClient) Call(data []byte, callback BinaryCallback) int {
	return 0
}

func (c *NativeClient) Close() {

}

var lock sync.Mutex
var clients map[uintptr]*NativeClient
var nextClientID uintptr

func init() {
	clients = make(map[uintptr]*NativeClient)
}

// kiconnect_sdk_new creates sdk handler.
// userData must be allocated on HEAP
//
//export kiconnect_sdk_new
func kiconnect_sdk_new(h *C.kiconnect_sdk_handler_t, data unsafe.Pointer, length C.int, app_callback C.kiconnect_sdk_application_callback, app_cbk_user_data unsafe.Pointer) C.int {
	appCbk := &CAppCallback{callback: app_callback, userData: app_cbk_user_data}
	client, err := NewClient(C.GoBytes(data, length), appCbk)
	if err != nil {
		return C.kiconnect_sdk_ERROR
	}

	lock.Lock()
	defer lock.Unlock()
	clientID := nextClientID
	clients[nextClientID] = client
	nextClientID++

	*h = C.kiconnect_sdk_handler_t(uintptr(clientID))

	return C.kiconnect_sdk_OK
}

func popClient(clientID uintptr) (client *NativeClient, ok bool) {
	lock.Lock()
	defer lock.Unlock()
	client, ok = clients[clientID]
	return
}

//export kiconnect_sdk_free
func kiconnect_sdk_free(h C.kiconnect_sdk_handler_t) {
	clientID := uintptr(h)

	client, ok := popClient(clientID)
	if !ok {
		return
	}
	client.Close()
}

// The call function is an exported API which is routing based on the type of the data argument:
//
//	static extern int call(byte[] data, int length, ResponseCallback callback, void* userData);
//
// userData must be allocated on HEAP
// It returns 0 if the request has been parsed successfully.
// It returns -1 otherwise.
//
//export kiconnect_sdk_call
func kiconnect_sdk_call(h C.kiconnect_sdk_handler_t, data unsafe.Pointer, length C.int, callback C.ResponseCallback, user_data unsafe.Pointer) C.int {
	clientID := uintptr(h)

	lock.Lock()
	client, ok := clients[clientID]
	lock.Unlock()
	if !ok {
		return C.kiconnect_sdk_INVALID_ARG
	}

	return C.int(client.Call(C.GoBytes(data, length), CCallback{callback: callback, userData: user_data}))
}

// CCallback wraps the C function reference.
type CCallback struct {
	callback C.ResponseCallback
	userData unsafe.Pointer
}

// Respond calls the underlying C function reference.
func (c CCallback) Respond(data []byte) {
	C.bridgeResponseCallback(c.callback, C.CBytes(data), C.size_t(len(data)), c.userData)
}

type CAppCallback struct {
	callback C.kiconnect_sdk_application_callback
	userData unsafe.Pointer
}

func errCodeToError(errCode C.int) error {
	switch errCode {
	case C.kiconnect_sdk_OK:
		return nil
	case C.kiconnect_sdk_ERROR:
		return fmt.Errorf("generic error")
	case C.kiconnect_sdk_INVALID_ARG:
		return fmt.Errorf("invalid argument")
	case C.kiconnect_sdk_INSUFFICIENT_BUFFER:
		return fmt.Errorf("insufficient buffer")
	case C.kiconnect_sdk_NOT_SUPPORTED:
		return fmt.Errorf("not supported")
	}
	return fmt.Errorf("unknown error code %v", int(errCode))
}

func (c *CAppCallback) CallApplication(request []byte) ([]byte, error) {
	req := C.CBytes(request)
	defer C.free(req)
	req_len := C.size_t(len(request))

	resp_len := C.size_t(0)
	resp_size := C.size_t(4096)
	resp := C.malloc(resp_size)
	if resp == nil {
		return nil, fmt.Errorf("cannot allocate memory")
	}

	for {
		errCode := C.bridgeApplicationCallback(c.callback, req, req_len, resp, resp_size, &resp_len, c.userData)
		switch errCode {
		case C.kiconnect_sdk_OK:
			data := C.GoBytes(resp, C.int(resp_len))
			C.free(resp)
			return data, nil
		case C.kiconnect_sdk_INSUFFICIENT_BUFFER:
			resp_size = resp_len
			tmp := C.realloc(resp, resp_size)
			if tmp == nil {
				return nil, fmt.Errorf("cannot allocate memory")
			}
			resp = tmp
		default:
			C.free(resp)
			return nil, errCodeToError(errCode)
		}
	}
}

func main() {}
