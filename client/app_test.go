package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/3scale/3scale-porta-go-client/fake"
)

func TestCreateApp(t *testing.T) {
	const (
		credential = "123"
		accountID  = "321"
		planID     = "123"
		name       = "test"
	)

	inputs := []struct {
		name       string
		returnErr  bool
		expectCode int
		expectErr  string
	}{
		{
			name:      "Test app creation fail",
			returnErr: true,
			expectErr: `error calling 3scale system - reason: { "error": "Your access token does not have the correct permissions" } - code: 403`,
		},
		{
			name: "Test app creation success",
		},
	}

	for _, input := range inputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			if req.Method != http.MethodPost {
				t.Fatalf("wrong helper called for create app api")
			}

			if req.URL.Path != "/admin/api/accounts/321/applications.json" {
				t.Fatal("wrong url generated by CreateApp function")
			}

			if input.returnErr {
				return fake.CreateAppError()
			}
			return fake.CreateAppSuccess(input.name)
		})

		c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)

		t.Run(input.name, func(t *testing.T) {
			a, b := c.CreateApp(accountID, planID, name, input.name)
			if input.returnErr {
				e := b.(ApiErr)
				if e.Code() != http.StatusForbidden {
					t.Fatal("unexpected code returned in error")
				}
				if b.Error() != input.expectErr {
					t.Fatalf("unexpected error message. Error received: %s", b.Error())
				}
				return
			}

			if a.Error != "" {
				t.Fatal("expected error to be empty")
			}
			if a.Description != input.name {
				t.Fatal("xml has not decoded correctly")
			}
		})
	}
}

func TestListApp(t *testing.T) {
	const (
		accessToken = "someAccessToken"
		accountID   = int64(321)
	)

	inputs := []struct {
		Name             string
		ExpectErr        bool
		ResponseCode     int
		ResponseBodyFile string
		ExpectedErrorMsg string
	}{
		{
			Name:             "ListAppOK",
			ExpectErr:        false,
			ResponseCode:     200,
			ResponseBodyFile: "app_list_response_fixture.json",
			ExpectedErrorMsg: "",
		},
		{
			Name:             "ListAppErr",
			ExpectErr:        true,
			ResponseCode:     400,
			ResponseBodyFile: "error_response_fixture.json",
			ExpectedErrorMsg: "Test Error",
		},
	}

	for _, input := range inputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			if req.Method != http.MethodGet {
				t.Fatalf("wrong helper called")
			}

			if req.URL.Path != fmt.Sprintf(appList, accountID) {
				t.Fatalf("wrong url generated")
			}

			bodyReader := bytes.NewReader(helperLoadBytes(t, input.ResponseBodyFile))
			return &http.Response{
				StatusCode: input.ResponseCode,
				Body:       ioutil.NopCloser(bodyReader),
				Header:     make(http.Header),
			}
		})

		c := NewThreeScale(NewTestAdminPortal(t), accessToken, httpClient)

		t.Run(input.Name, func(subTest *testing.T) {
			appList, err := c.ListApplications(accountID)
			if input.ExpectErr {
				if err == nil {
					subTest.Fatalf("client operation did not return error")
				}

				apiError, ok := err.(ApiErr)
				if !ok {
					subTest.Fatalf("expected ApiErr error type")
				}

				if !strings.Contains(apiError.Error(), input.ExpectedErrorMsg) {
					subTest.Fatalf("Expected [%s]: got [%s] ", input.ExpectedErrorMsg, apiError.Error())
				}

			} else {
				if err != nil {
					subTest.Fatal(err)
				}
				if appList == nil {
					subTest.Fatalf("appList not parsed")
				}
				if len(appList.Applications) == 0 {
					subTest.Fatalf("appList empty")
				}
				if appList.Applications[0].Application.ID != 146 {
					subTest.Fatalf("appList not parsed")
				}
			}
		})
	}
}

func TestDeleteApp(t *testing.T) {
	var (
		accessToken       = "someAccessToken"
		accountID   int64 = 321
		appID       int64 = 21
		endpoint          = fmt.Sprintf(appDelete, accountID, appID)
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {

		if req.URL.Path != endpoint {
			t.Fatal("wrong url generated by CreateApp function")
		}

		if req.Method != http.MethodDelete {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodDelete, req.Method)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}
	})

	c := NewThreeScale(NewTestAdminPortal(t), accessToken, httpClient)
	err := c.DeleteApplication(accountID, appID)
	if err != nil {
		t.Fatal(err)
	}

}

func TestUpdateApplication(t *testing.T) {
	var (
		appID     int64 = 12
		accountID int64 = 321
		params          = Params{"": "newDescription "}
		endpoint        = fmt.Sprintf(appUpdate, accountID, appID)
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != endpoint {
			t.Fatalf("Path does not match. Expected [%s]; got [%s]", endpoint, req.URL.Path)
		}

		if req.Method != http.MethodPut {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodPut, req.Method)
		}

		application := &ApplicationElem{
			Application{
				UserAccountID: strconv.FormatInt(accountID, 10),
				ID:            appID,
				AppName:       "newName",
			},
		}
		responseBodyBytes, err := json.Marshal(application)
		if err != nil {
			t.Fatal(err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes)),
			Header:     make(http.Header),
		}
	})

	credential := "someAccessToken"
	c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)
	obj, err := c.UpdateApplication(accountID, appID, params)
	if err != nil {
		t.Fatal(err)
	}

	if obj == nil {
		t.Fatal("application returned nil")
	}

	if obj.ID != appID {
		t.Fatalf("obj ID does not match. Expected [%d]; got [%d]", appID, obj.ID)
	}

	if obj.AppName != "newName" {
		t.Fatalf("obj name does not match. Expected [%s]; got [%s]", "newName", obj.AppName)
	}
}

func TestChangeApplicationPlan(t *testing.T) {
	var (
		appID     int64 = 12
		accountID int64 = 321
		planID    int64 = 14
		endpoint        = fmt.Sprintf(appChangePlan, accountID, appID)
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != endpoint {
			t.Fatalf("Path does not match. Expected [%s]; got [%s]", endpoint, req.URL.Path)
		}

		if req.Method != http.MethodPut {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodPut, req.Method)
		}

		application := &ApplicationElem{
			Application{
				UserAccountID: strconv.FormatInt(accountID, 10),
				ID:            appID,
				PlanID:        16,
			},
		}
		responseBodyBytes, err := json.Marshal(application)
		if err != nil {
			t.Fatal(err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes)),
			Header:     make(http.Header),
		}
	})

	credential := "someAccessToken"
	c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)
	obj, err := c.ChangeApplicationPlan(accountID, appID, planID)
	if err != nil {
		t.Fatal(err)
	}

	if obj == nil {
		t.Fatal("application returned nil")
	}

	if obj.ID != appID {
		t.Fatalf("obj ID does not match. Expected [%d]; got [%d]", appID, obj.ID)
	}

	if obj.PlanID != 16 {
		t.Fatalf("obj name does not match. Expected [%d]; got [%d]", 16, obj.PlanID)
	}
}

func TestCreateApplicationCustomPlan(t *testing.T) {
	var (
		appID      int64 = 12
		accountID  int64 = 321
		ID         int64 = 21
		Name             = "customPlan"
		SystemName       = "customPlan"
		Custom           = true
		endpoint         = fmt.Sprintf(appCreatePlanCustomization, accountID, appID)
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != endpoint {
			t.Fatalf("Path does not match. Expected [%s]; got [%s]", endpoint, req.URL.Path)
		}

		if req.Method != http.MethodPut {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodPut, req.Method)
		}

		applicationPlan := &ApplicationPlan{
			Element: ApplicationPlanItem{
				ID:         ID,
				Name:       Name,
				SystemName: SystemName,
				Custom:     Custom,
			},
		}

		responseBodyBytes, err := json.Marshal(applicationPlan)
		if err != nil {
			t.Fatal(err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes)),
			Header:     make(http.Header),
		}
	})

	credential := "someAccessToken"
	c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)
	obj, err := c.CreateApplicationCustomPlan(accountID, appID)
	if err != nil {
		t.Fatal(err)
	}

	if obj == nil {
		t.Fatal("CreateCustomPlan returned nil")
	}

	if obj.ID != ID {
		t.Fatalf("obj ID does not match. Expected [%d]; got [%d]", ID, obj.ID)
	}

	if obj.Name != Name {
		t.Fatalf("obj name does not match. Expected [%s]; got [%s]", Name, obj.Name)
	}

	if obj.SystemName != SystemName {
		t.Fatalf("obj system name does not match. Expected [%s]; got [%s]", SystemName, obj.SystemName)
	}

	if obj.Custom != true {
		t.Fatalf("obj custom bool does not match. Expected [%t]; got [%t]", true, obj.Custom)
	}
}

func TestDeleteApplicationCustomPlan(t *testing.T) {
	var (
		accountID int64 = 321
		appID     int64 = 12
		endpoint        = fmt.Sprintf(appDeletePlanCustomization, accountID, appID)
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != endpoint {
			t.Fatalf("Path does not match. Expected [%s]; got [%s]", endpoint, req.URL.Path)
		}

		if req.Method != http.MethodPut {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodPut, req.Method)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}
	})

	credential := "someAccessToken"
	c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)
	err := c.DeleteApplicationCustomPlan(accountID, appID)
	if err != nil {
		t.Fatal(err)
	}
}

func TestApplicationSuspend(t *testing.T) {
	var (
		appID     int64 = 12
		accountID int64 = 321
		state           = "suspended"
		endpoint        = fmt.Sprintf(appSuspend, accountID, appID)
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != endpoint {
			t.Fatalf("Path does not match. Expected [%s]; got [%s]", endpoint, req.URL.Path)
		}

		if req.Method != http.MethodPut {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodPut, req.Method)
		}

		application := &ApplicationElem{
			Application{
				ID:            appID,
				UserAccountID: strconv.FormatInt(accountID, 10),
				State:         state,
			},
		}
		responseBodyBytes, err := json.Marshal(application)
		if err != nil {
			t.Fatal(err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes)),
			Header:     make(http.Header),
		}
	})

	credential := "someAccessToken"
	c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)
	obj, err := c.ApplicationSuspend(accountID, appID)
	if err != nil {
		t.Fatal(err)
	}

	if obj == nil {
		t.Fatal("application returned nil")
	}

	if obj.State != state {
		t.Fatalf("obj state does not match. Expected [%d]; got [%d]", appID, obj.ID)
	}
}

func TestApplicationResume(t *testing.T) {
	var (
		appID     int64 = 12
		accountID int64 = 321
		state           = "Live"
		endpoint        = fmt.Sprintf(appResume, accountID, appID)
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != endpoint {
			t.Fatalf("Path does not match. Expected [%s]; got [%s]", endpoint, req.URL.Path)
		}

		if req.Method != http.MethodPut {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodPut, req.Method)
		}

		application := &ApplicationElem{
			Application{
				ID:            appID,
				UserAccountID: strconv.FormatInt(accountID, 10),
				State:         state,
			},
		}
		responseBodyBytes, err := json.Marshal(application)
		if err != nil {
			t.Fatal(err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes)),
			Header:     make(http.Header),
		}
	})

	credential := "someAccessToken"
	c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)
	obj, err := c.ApplicationResume(accountID, appID)
	if err != nil {
		t.Fatal(err)
	}

	if obj == nil {
		t.Fatal("application returned nil")
	}

	if obj.State != state {
		t.Fatalf("obj state does not match. Expected [%d]; got [%d]", appID, obj.ID)
	}
}

func TestReadApplication(t *testing.T) {
	var (
		ID          int64 = 987
		accountID   int64 = 98765
		planID      int64 = 21
		description       = "description"
		endpoint          = fmt.Sprintf(appRead, accountID, ID)
		application       = &ApplicationElem{
			Application{
				ID:            ID,
				PlanID:        planID,
				UserAccountID: strconv.FormatInt(accountID, 10),
				Description:   description,
				ApplicationId: "7034ff61",
			},
		}
	)

	httpClient := NewTestClient(func(req *http.Request) *http.Response {
		if req.URL.Path != endpoint {
			t.Fatalf("Path does not match. Expected [%s]; got [%s]", endpoint, req.URL.Path)
		}

		if req.Method != http.MethodGet {
			t.Fatalf("Method does not match. Expected [%s]; got [%s]", http.MethodGet, req.Method)
		}

		responseBodyBytes, err := json.Marshal(application)
		if err != nil {
			t.Fatal(err)
		}

		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       ioutil.NopCloser(bytes.NewBuffer(responseBodyBytes)),
			Header:     make(http.Header),
		}
	})

	credential := "someAccessToken"
	c := NewThreeScale(NewTestAdminPortal(t), credential, httpClient)
	obj, err := c.Application(accountID, ID)
	if err != nil {
		t.Fatal(err)
	}

	if obj == nil {
		t.Fatal("application returned nil")
	}

	if *obj != application.Application {
		t.Fatalf("Expected %v; got %v", application, *obj)
	}
}

func TestListAllApplications(t *testing.T) {
	const (
		accessToken = "someAccessToken"
	)

	inputs := []struct {
		Name             string
		ExpectErr        bool
		ResponseCode     int
		ResponseBodyFile string
		ExpectedErrorMsg string
	}{
		{
			Name:             "ListAppOK",
			ExpectErr:        false,
			ResponseCode:     200,
			ResponseBodyFile: "app_list_response_fixture.json",
			ExpectedErrorMsg: "",
		},
		{
			Name:             "ListAppErr",
			ExpectErr:        true,
			ResponseCode:     400,
			ResponseBodyFile: "error_response_fixture.json",
			ExpectedErrorMsg: "Test Error",
		},
	}

	for _, input := range inputs {
		httpClient := NewTestClient(func(req *http.Request) *http.Response {
			if req.Method != http.MethodGet {
				t.Fatalf("wrong helper called")
			}

			if req.URL.Path != listAllApplications {
				t.Fatalf("wrong url generated")
			}

			bodyReader := bytes.NewReader(helperLoadBytes(t, input.ResponseBodyFile))
			return &http.Response{
				StatusCode: input.ResponseCode,
				Body:       ioutil.NopCloser(bodyReader),
				Header:     make(http.Header),
			}
		})

		c := NewThreeScale(NewTestAdminPortal(t), accessToken, httpClient)

		t.Run(input.Name, func(subTest *testing.T) {
			appList, err := c.ListAllApplications()
			if input.ExpectErr {
				if err == nil {
					subTest.Fatalf("client operation did not return error")
				}

				apiError, ok := err.(ApiErr)
				if !ok {
					subTest.Fatalf("expected ApiErr error type")
				}

				if !strings.Contains(apiError.Error(), input.ExpectedErrorMsg) {
					subTest.Fatalf("Expected [%s]: got [%s] ", input.ExpectedErrorMsg, apiError.Error())
				}

			} else {
				if err != nil {
					subTest.Fatal(err)
				}
				if appList == nil {
					subTest.Fatalf("appList not parsed")
				}
				if len(appList.Applications) == 0 {
					subTest.Fatalf("appList empty")
				}
				if appList.Applications[0].Application.ID != 146 {
					subTest.Fatalf("appList not parsed")
				}
			}
		})
	}
}
