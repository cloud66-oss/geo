package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/cloud66-oss/geo/cache"
	"github.com/cloud66-oss/geo/provider"
	"github.com/cloud66-oss/geo/utils"
	"github.com/labstack/echo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type mockProvider struct {
	mock.Mock
}

var _ provider.IPProvider = &mockProvider{}

type mockCacheProvider struct {
	mock.Mock
}

var _ cache.CacheProvider = &mockCacheProvider{}

type serveCmdTestSuite struct {
	suite.Suite
	provider *mockProvider
	cache    *mockCacheProvider
}

func (mp *mockProvider) Start(ctx context.Context) error {
	args := mp.Called(ctx)
	return args.Error(0)
}

func (mp *mockProvider) Lookup(ctx context.Context, address string, asFallback bool) (*utils.IPInfo, error) {
	args := mp.Called(ctx, address)

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

func (mcp *mockCacheProvider) Fetch(ctx context.Context, provider string, address string) (*utils.IPInfo, error) {
	args := mcp.Called(ctx, provider, address)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*utils.IPInfo), args.Error(1)
}

func (mcp *mockCacheProvider) Add(ctx context.Context, provider string, ipInfo *utils.IPInfo) error {
	args := mcp.Called(ctx, provider, ipInfo)
	return args.Error(0)
}

func (suite *serveCmdTestSuite) SetupTest() {
	ctx := context.Background()
	utils.Container.Clear(ctx)

	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	log.Logger = log.With().Caller().Logger()
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})

	suite.provider = &mockProvider{}
	suite.cache = &mockCacheProvider{}
	utils.Container.Assign(ctx, utils.MaxMindProvider, suite.provider)
	utils.Container.Assign(ctx, utils.Cache, suite.cache)
}

func (suite *serveCmdTestSuite) TestPing() {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/_ping")

	if suite.Assert().NoError(ping(c)) {
		suite.Assert().EqualValues(http.StatusOK, rec.Code)
	}
}

func (suite *serveCmdTestSuite) TestLookupWithoutCache() {
	viper.Set("cache.enabled", false)

	suite.provider.On("Lookup", mock.Anything, "1.1.1.1").Return(&utils.IPInfo{Address: "1.1.1.1"}, nil)
	suite.cache.On("Add", mock.Anything, "1.1.1.1").Return(nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/v1/ip/:address")
	c.SetParamNames("address")
	c.SetParamValues("1.1.1.1")

	if suite.Assert().NoError(getIP(c)) {
		suite.Assert().EqualValues(http.StatusOK, rec.Code)
		suite.cache.AssertNotCalled(suite.T(), "Fetch", mock.Anything, mock.Anything, mock.Anything)
		suite.cache.AssertNotCalled(suite.T(), "Add", mock.Anything, mock.Anything, mock.Anything)
	}
}

func (suite *serveCmdTestSuite) TestLookupWithCache() {
	viper.Set("cache.enabled", true)

	suite.provider.On("Lookup", mock.Anything, "1.1.1.1").Return(&utils.IPInfo{Address: "1.1.1.1"}, nil)
	suite.cache.On("Fetch", mock.Anything, "maxmind", "1.1.1.1").Return(&utils.IPInfo{Address: "1.1.1.1"}, nil)
	suite.cache.On("Add", mock.Anything, "maxmind", mock.Anything).Return(nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/v1/ip/:address")
	c.SetParamNames("address")
	c.SetParamValues("1.1.1.1")

	if suite.Assert().NoError(getIP(c)) {
		suite.Assert().EqualValues(http.StatusOK, rec.Code)
		suite.cache.AssertNumberOfCalls(suite.T(), "Fetch", 1)
		suite.cache.AssertNotCalled(suite.T(), "Add", mock.Anything)
	}
}

func (suite *serveCmdTestSuite) TestLookupWithEmptyCache() {
	viper.Set("cache.enabled", true)

	suite.provider.On("Lookup", mock.Anything, "2.2.2.2").Return(&utils.IPInfo{Address: "2.2.2.2"}, nil)
	suite.cache.On("Fetch", mock.Anything, "maxmind", "2.2.2.2").Return(nil, nil)
	suite.cache.On("Add", mock.Anything, "maxmind", mock.Anything).Return(nil)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/v1/ip/:address")
	c.SetParamNames("address")
	c.SetParamValues("2.2.2.2")

	if suite.Assert().NoError(getIP(c)) {
		suite.Assert().EqualValues(http.StatusOK, rec.Code)
		suite.cache.AssertNumberOfCalls(suite.T(), "Fetch", 1)
		suite.cache.AssertCalled(suite.T(), "Add", mock.Anything, "maxmind", mock.MatchedBy(func(ipInfo *utils.IPInfo) bool { return ipInfo.Address == "2.2.2.2" }))
	}
}
func TestServeCmdTestSuite(t *testing.T) {
	suite.Run(t, new(serveCmdTestSuite))
}
