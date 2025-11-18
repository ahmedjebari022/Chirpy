package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)





func TestHashing(t *testing.T){
	testCase := []struct{
		input string
		password string
		want bool
	}{
		{input: "hello", password: "hello" , want: true},
		{input: "hello", password: "hi" , want: false},
	}

	for i,c := range testCase{
		hash, _ := HashPassword(c.password)
		match, _ := CheckPasswordHash(c.input,hash)
		if match != c.want{
			t.Errorf(`Test Case :%d FAILED input: %s password: %s wanted: %t got: %t`,i,c.input,c.password,c.want,match)
		}
	}
}


func TestJWT(t *testing.T){
	inputId := uuid.New()
	testCase := []struct{
		input string
		wait time.Duration
		want uuid.UUID
	}{
		{input: "secret_key", want:inputId, wait: time.Duration(1*time.Second)},
		{input: "not secret_key" , want:uuid.UUID{}, wait: time.Duration(1*time.Second)},
		{input: "secret_key" , want:uuid.UUID{}, wait: time.Duration(4*time.Second)},
	}
	
	for i, c := range testCase{
		jwt, _ := MakeJWT(inputId,"secret_key",2*time.Second)
		time.Sleep(c.wait)
		id, _ := ValidateJWT(jwt,c.input)
		if id != c.want{
			t.Errorf("error at case: %d result: %s wanted: %s",i,id,c.want)
		}
	}

}

func TestGetBearerToken(t *testing.T){
	testCase := []struct{
		input string
		want string
	}{
		{input: "token",want: "token"},
		{input: "no_token",want: "no_token"},
		{input: "",want: ""},
	}
	for i,c := range testCase{
		req := http.Request{
			Header: make(map[string][]string,0),
		}
		req.Header.Set("Authorization","Bearer "+c.input)
		token,err := GetBearerToken(req.Header)
		if err != nil {
			fmt.Printf("err: %s",err.Error())
		}
		if token != c.want{
			t.Errorf("Failed case :%d expected: %s got: %s\n",i,c.want,token)
		}
	}


}