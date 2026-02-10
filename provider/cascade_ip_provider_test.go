package provider

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/cloud66-oss/geo/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockProvider struct {
	mock.Mock
}

type cascadeIpProviderTestSuite struct {
	suite.Suite
}

func (mp *mockProvider) Start(ctx context.Context) error {
	args := mp.Called(ctx)
	return args.Error(0)
}

func (mp *mockProvider) Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error) {
	args := mp.Called(ctx, address, asFallback)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*utils.IPInfo), args.Error(1)
}

func (mp *mockProvider) Shutdown(ctx context.Context) {
	mp.Called(ctx)
}

func (mp *mockProvider) Refresh(ctx context.Context) error {
	args := mp.Called(ctx)
	return args.Error(0)
}

func (suite *cascadeIpProviderTestSuite) SetupTest() {
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	log.Logger = log.With().Caller().Logger()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})
}

func (suite *cascadeIpProviderTestSuite) TestCascadeFlowFirstReturn() {
	ctx := context.Background()

	p1 := &mockProvider{}
	p1.On("Lookup", ctx, "1.1.1.1", false).Return(&utils.IPInfo{Address: "1.1.1.1"}, nil)
	p2 := &mockProvider{}
	p2.AssertNotCalled(suite.T(), "Lookup", mock.Anything)

	provider, err := NewCascadeIPProvider(ctx, true, []IPProvider{p1, p2})
	suite.NoError(err)
	info, err := provider.Lookup(ctx, "1.1.1.1", false)
	suite.NoError(err)

	p1.AssertExpectations(suite.T())
	p2.AssertExpectations(suite.T())

	suite.EqualValues("1.1.1.1", info.Address)
}

func (suite *cascadeIpProviderTestSuite) TestCascadeFlowSecondReturn() {
	ctx := context.Background()

	p1 := &mockProvider{}
	p1.On("Lookup", ctx, "1.1.1.1", false).Return(nil, nil)
	p2 := &mockProvider{}
	p2.On("Lookup", ctx, "1.1.1.1", true).Return(&utils.IPInfo{Address: "2.2.2.2"}, nil)

	provider, err := NewCascadeIPProvider(ctx, true, []IPProvider{p1, p2})
	suite.NoError(err)
	info, err := provider.Lookup(ctx, "1.1.1.1", false)
	suite.NoError(err)

	p1.AssertExpectations(suite.T())
	p2.AssertExpectations(suite.T())

	suite.EqualValues("2.2.2.2", info.Address)
}

func (suite *cascadeIpProviderTestSuite) TestCascadeFlowErrorReturn() {
	ctx := context.Background()

	p1 := &mockProvider{}
	p1.On("Lookup", ctx, "1.1.1.1", false).Return(nil, errors.New("something broke"))
	p2 := &mockProvider{}
	p2.AssertNotCalled(suite.T(), mock.Anything)

	provider, err := NewCascadeIPProvider(ctx, true, []IPProvider{p1, p2})
	suite.NoError(err)
	_, err = provider.Lookup(ctx, "1.1.1.1", false)

	p1.AssertExpectations(suite.T())
	p2.AssertExpectations(suite.T())

	suite.Assert().Error(err, "something broken")
}

func TestCascadeIpProviderTestSuite(t *testing.T) {
	suite.Run(t, new(cascadeIpProviderTestSuite))
}
