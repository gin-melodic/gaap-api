package utils_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"gaap-api/internal/logic/utils"

	"github.com/gogf/gf/v2/test/gtest"
)

// Test_GetOrLoad_Fallback tests that GetOrLoad falls back to loader when Redis is unavailable.
// Since Redis is not configured in test environment, this verifies graceful degradation.
func Test_GetOrLoad_Fallback(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()

		// Test with a simple string type
		result, err := utils.GetOrLoad(
			ctx,
			"test:key:1",
			time.Minute,
			func(ctx context.Context) (*string, error) {
				value := "test_value"
				return &value, nil
			},
		)

		g.AssertNil(err)
		g.AssertNE(result, nil)
		g.Assert(*result, "test_value")
	})
}

// Test_GetOrLoad_LoaderError tests that GetOrLoad properly propagates loader errors.
func Test_GetOrLoad_LoaderError(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()
		expectedErr := errors.New("loader error")

		result, err := utils.GetOrLoad(
			ctx,
			"test:key:2",
			time.Minute,
			func(ctx context.Context) (*string, error) {
				return nil, expectedErr
			},
		)

		g.AssertNE(err, nil)
		g.Assert(result, nil)
	})
}

// Test_GetOrLoad_NilResult tests that GetOrLoad handles nil result from loader.
func Test_GetOrLoad_NilResult(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()

		result, err := utils.GetOrLoad(
			ctx,
			"test:key:3",
			time.Minute,
			func(ctx context.Context) (*string, error) {
				return nil, nil
			},
		)

		g.AssertNil(err)
		g.Assert(result, nil)
	})
}

// TestStruct is a test struct for complex type caching
type TestStruct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Test_GetOrLoad_ComplexType tests caching with complex struct types.
func Test_GetOrLoad_ComplexType(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()

		result, err := utils.GetOrLoad(
			ctx,
			"test:struct:1",
			time.Minute,
			func(ctx context.Context) (*TestStruct, error) {
				return &TestStruct{ID: 1, Name: "test"}, nil
			},
		)

		g.AssertNil(err)
		g.AssertNE(result, nil)
		g.Assert(result.ID, 1)
		g.Assert(result.Name, "test")
	})
}

// Test_BatchGetOrLoad_Fallback tests that BatchGetOrLoad falls back to loader.
func Test_BatchGetOrLoad_Fallback(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()
		ids := []interface{}{1, 2, 3}

		results, err := utils.BatchGetOrLoad(
			ctx,
			ids,
			"test:batch",
			time.Minute,
			func(ts *TestStruct) interface{} {
				return ts.ID
			},
			func(ctx context.Context, missedIDs []interface{}) ([]*TestStruct, error) {
				// All IDs should be missed since Redis is unavailable
				g.Assert(len(missedIDs), 3)

				items := make([]*TestStruct, len(missedIDs))
				for i, id := range missedIDs {
					items[i] = &TestStruct{ID: id.(int), Name: "test"}
				}
				return items, nil
			},
		)

		g.AssertNil(err)
		g.Assert(len(results), 3)
	})
}

// Test_BatchGetOrLoad_EmptyIds tests BatchGetOrLoad with empty IDs.
func Test_BatchGetOrLoad_EmptyIds(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()
		ids := []interface{}{}

		results, err := utils.BatchGetOrLoad(
			ctx,
			ids,
			"test:batch",
			time.Minute,
			func(ts *TestStruct) interface{} {
				return ts.ID
			},
			func(ctx context.Context, missedIDs []interface{}) ([]*TestStruct, error) {
				// Should not be called for empty IDs
				g.Error("Loader should not be called for empty IDs")
				return nil, nil
			},
		)

		g.AssertNil(err)
		g.Assert(len(results), 0)
	})
}

// Test_InvalidateCache_NoRedis tests that InvalidateCache doesn't error when Redis unavailable.
func Test_InvalidateCache_NoRedis(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()

		// Should not return error even when Redis is unavailable
		err := utils.InvalidateCache(ctx, "test:key:1", "test:key:2")
		g.AssertNil(err)
	})
}

// Test_InvalidatePattern_NoRedis tests that InvalidatePattern doesn't error when Redis unavailable.
func Test_InvalidatePattern_NoRedis(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		ctx := context.Background()

		// Should not return error even when Redis is unavailable
		err := utils.InvalidatePattern(ctx, "test:*")
		g.AssertNil(err)
	})
}

// Test_UserCacheKey tests the UserCacheKey helper function.
func Test_UserCacheKey(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		key := utils.UserCacheKey("user-123")
		g.Assert(key, "user:user-123")

		key = utils.UserCacheKey("abc-def-ghi")
		g.Assert(key, "user:abc-def-ghi")
	})
}

// Test_CacheTTL_Defaults tests that CacheTTL has reasonable default values.
func Test_CacheTTL_Defaults(t *testing.T) {
	gtest.C(t, func(g *gtest.T) {
		// Verify defaults are set
		g.Assert(utils.CacheTTL.Config > 0, true)
		g.Assert(utils.CacheTTL.User > 0, true)
		g.Assert(utils.CacheTTL.Account > 0, true)
		g.Assert(utils.CacheTTL.Transaction > 0, true)
		g.Assert(utils.CacheTTL.Dashboard > 0, true)
		g.Assert(utils.CacheTTL.Search > 0, true)

		// Verify reasonable values (Config should be longer than Dashboard)
		g.Assert(utils.CacheTTL.Config > utils.CacheTTL.Dashboard, true)
	})
}
