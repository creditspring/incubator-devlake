/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/stretchr/testify/assert"
)

func fakeResponse(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Request:    &http.Request{URL: &url.URL{Path: "/test"}},
	}
}

func TestBasecampResponseHook_404_Skips(t *testing.T) {
	hook := basecampResponseHook(&nopLogger{})
	assert.Equal(t, api.ErrIgnoreAndContinue, hook(fakeResponse(http.StatusNotFound)))
}

func TestBasecampResponseHook_401_Skips(t *testing.T) {
	hook := basecampResponseHook(&nopLogger{})
	assert.Equal(t, api.ErrIgnoreAndContinue, hook(fakeResponse(http.StatusUnauthorized)))
}

func TestBasecampResponseHook_200_PassesThrough(t *testing.T) {
	hook := basecampResponseHook(&nopLogger{})
	assert.Nil(t, hook(fakeResponse(http.StatusOK)))
}

func TestBasecampResponseHook_500_PassesThrough(t *testing.T) {
	hook := basecampResponseHook(&nopLogger{})
	assert.Nil(t, hook(fakeResponse(http.StatusInternalServerError)))
}

func TestBasecampResponseHook_429_PassesThrough(t *testing.T) {
	hook := basecampResponseHook(&nopLogger{})
	assert.Nil(t, hook(fakeResponse(http.StatusTooManyRequests)))
}
