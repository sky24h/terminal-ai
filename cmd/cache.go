package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/terminal-ai/internal/ai"
)

var (
	clearCache        bool
	showStats         bool
	invalidatePattern string
)

// cacheCmd represents the cache command
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the response cache",
	Long: `Manage the AI response cache for improved performance.

The cache stores AI responses to avoid redundant API calls for identical queries.
It uses an LRU (Least Recently Used) eviction strategy and supports TTL.

Examples:
  terminal-ai cache --stats           # Show cache statistics
  terminal-ai cache --clear           # Clear all cached responses
  terminal-ai cache --invalidate "chat_"  # Invalidate entries matching pattern`,
	RunE: runCache,
}

func init() {
	rootCmd.AddCommand(cacheCmd)

	// Cache command flags
	cacheCmd.Flags().BoolVarP(&clearCache, "clear", "c", false, "clear all cached responses")
	cacheCmd.Flags().BoolVarP(&showStats, "stats", "s", false, "show cache statistics")
	cacheCmd.Flags().StringVarP(&invalidatePattern, "invalidate", "i", "", "invalidate cache entries matching pattern")
}

func runCache(cmd *cobra.Command, args []string) error {
	// Check if cache is enabled
	if !appConfig.Cache.Enabled {
		fmt.Println("Cache is disabled. Enable it in the configuration to use caching.")
		return nil
	}

	// Get the OpenAI client with cache
	client, ok := aiClient.(*ai.OpenAIClient)
	if !ok {
		return fmt.Errorf("cache operations not supported for this client type")
	}

	// Handle clear flag
	if clearCache {
		if err := client.ClearCache(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
		fmt.Println("✓ Cache cleared successfully")
		return nil
	}

	// Handle invalidate pattern
	if invalidatePattern != "" {
		count, err := client.InvalidateCachePattern(invalidatePattern)
		if err != nil {
			return fmt.Errorf("failed to invalidate cache pattern: %w", err)
		}
		fmt.Printf("✓ Invalidated %d cache entries matching pattern '%s'\n", count, invalidatePattern)
		return nil
	}

	// Show cache statistics (default or with --stats flag)
	stats := client.GetCacheStats()
	if stats == nil {
		fmt.Println("Cache statistics not available")
		return nil
	}

	displayCacheStats(stats)
	return nil
}

func displayCacheStats(stats *ai.CacheStats) {
	fmt.Println("Cache Statistics")
	fmt.Println("================")

	// Performance metrics
	fmt.Println("\nPerformance:")
	fmt.Printf("  Hit Rate:     %.1f%%\n", stats.HitRate*100)
	fmt.Printf("  Total Hits:   %d\n", stats.Hits)
	fmt.Printf("  Total Misses: %d\n", stats.Misses)

	// Storage metrics
	fmt.Println("\nStorage:")
	fmt.Printf("  Entries:      %d\n", stats.Entries)
	fmt.Printf("  Size:         %.2f MB / %.2f MB\n",
		float64(stats.SizeBytes)/(1024*1024),
		float64(stats.MaxSizeBytes)/(1024*1024))
	fmt.Printf("  Usage:        %.1f%%\n",
		float64(stats.SizeBytes)/float64(stats.MaxSizeBytes)*100)

	// Eviction metrics
	fmt.Println("\nEvictions:")
	fmt.Printf("  Total:        %d\n", stats.Evictions)

	// Last cleanup
	if !stats.LastCleanup.IsZero() {
		fmt.Printf("\nLast Cleanup: %s\n", stats.LastCleanup.Format("2006-01-02 15:04:05"))
	}

	// Configuration
	fmt.Println("\nConfiguration:")
	fmt.Printf("  Strategy:     %s\n", appConfig.Cache.Strategy)
	fmt.Printf("  TTL:          %s\n", appConfig.Cache.TTL)
	fmt.Printf("  Max Size:     %d MB\n", appConfig.Cache.MaxSize)

	if appConfig.Cache.Dir != "" {
		fmt.Printf("  Persist Dir:  %s\n", appConfig.Cache.Dir)

		// Check if cache file exists
		cachePath := appConfig.Cache.Dir + "/cache.gob"
		if info, err := os.Stat(cachePath); err == nil {
			fmt.Printf("  Cache File:   %.2f KB\n", float64(info.Size())/1024)
		}
	}

	// Efficiency summary
	fmt.Println("\nEfficiency Summary:")
	if stats.Hits+stats.Misses > 0 {
		avgSaving := calculateAverageSaving(stats)
		fmt.Printf("  Estimated API calls saved: %d\n", stats.Hits)
		fmt.Printf("  Estimated time saved: ~%.1f seconds\n", avgSaving)

		// Cost estimation (rough estimate based on GPT-3.5 pricing)
		estimatedCost := float64(stats.Hits) * 0.002 // Rough estimate: $0.002 per 1K tokens
		fmt.Printf("  Estimated cost saved: ~$%.2f\n", estimatedCost)
	}
}

func calculateAverageSaving(stats *ai.CacheStats) float64 {
	// Estimate average API call latency vs cache retrieval
	avgAPILatency := 1.5     // seconds (rough estimate)
	avgCacheLatency := 0.001 // seconds

	return float64(stats.Hits) * (avgAPILatency - avgCacheLatency)
}
