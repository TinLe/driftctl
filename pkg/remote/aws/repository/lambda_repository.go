package repository

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/lambda/lambdaiface"
)

type LambdaClient interface {
	lambdaiface.LambdaAPI
}

type LambdaRepository interface {
	ListAllLambdaFunctions() ([]*lambda.FunctionConfiguration, error)
}

type lambdaRepository struct {
	client lambdaiface.LambdaAPI
}

func NewLambdaRepository(session *session.Session) *lambdaRepository {
	return &lambdaRepository{
		lambda.New(session),
	}
}

func (r *lambdaRepository) ListAllLambdaFunctions() ([]*lambda.FunctionConfiguration, error) {
	var functions []*lambda.FunctionConfiguration
	input := &lambda.ListFunctionsInput{}
	err := r.client.ListFunctionsPages(input, func(res *lambda.ListFunctionsOutput, lastPage bool) bool {
		functions = append(functions, res.Functions...)
		return !lastPage
	})
	if err != nil {
		return nil, err
	}
	return functions, nil
}
