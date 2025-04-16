package optimizer

import (
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"
)

// SaveResultsToCSV saves optimization results to a CSV file
func SaveResultsToCSV(results []*Result, targetMetric MetricName, filePath string) error {
	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Sort results by the target metric
	sorter := ResultSorter{
		Results:    results,
		MetricName: string(targetMetric),
		Maximize:   true,
	}
	sort.Sort(sorter)

	// Determine all parameter names and metric names
	paramNames := make(map[string]bool)
	metricNames := make(map[string]bool)

	for _, result := range results {
		for paramName := range result.Parameters {
			paramNames[paramName] = true
		}
		for metricName := range result.Metrics {
			metricNames[metricName] = true
		}
	}

	// Convert to slices and sort for consistent ordering
	paramNameSlice := make([]string, 0, len(paramNames))
	for name := range paramNames {
		paramNameSlice = append(paramNameSlice, name)
	}
	sort.Strings(paramNameSlice)

	metricNameSlice := make([]string, 0, len(metricNames))
	for name := range metricNames {
		metricNameSlice = append(metricNameSlice, name)
	}
	sort.Strings(metricNameSlice)

	// Create header row
	header := []string{"Rank", "Duration"}
	header = append(header, paramNameSlice...)
	header = append(header, metricNameSlice...)
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	// Write each result
	for i, result := range results {
		row := []string{
			strconv.Itoa(i + 1),
			result.Duration.String(),
		}

		// Add parameter values
		for _, paramName := range paramNameSlice {
			value, exists := result.Parameters[paramName]
			if !exists {
				row = append(row, "")
				continue
			}

			// Convert value to string
			switch v := value.(type) {
			case int:
				row = append(row, strconv.Itoa(v))
			case float64:
				row = append(row, strconv.FormatFloat(v, 'f', 4, 64))
			case bool:
				row = append(row, strconv.FormatBool(v))
			case string:
				row = append(row, v)
			default:
				row = append(row, fmt.Sprintf("%v", v))
			}
		}

		// Add metric values
		for _, metricName := range metricNameSlice {
			value, exists := result.Metrics[metricName]
			if !exists {
				row = append(row, "")
				continue
			}
			row = append(row, strconv.FormatFloat(value, 'f', 4, 64))
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	return nil
}

// PrintResults prints optimization results to the console
func PrintResults(results []*Result, targetMetric MetricName, topN int) {
	if len(results) == 0 {
		fmt.Println("No results to display")
		return
	}

	// Sort results by the target metric
	sorter := ResultSorter{
		Results:    results,
		MetricName: string(targetMetric),
		Maximize:   true,
	}
	sort.Sort(sorter)

	// Limit to top N results
	if topN > 0 && topN < len(results) {
		results = results[:topN]
	}

	fmt.Printf("\n=== Top %d Results (by %s) ===\n\n", len(results), targetMetric)

	for i, result := range results {
		fmt.Printf("Rank #%d (Duration: %s)\n", i+1, result.Duration.Round(time.Millisecond))
		
		fmt.Println("Parameters:")
		for name, value := range result.Parameters {
			fmt.Printf("  %s: %v\n", name, value)
		}
		
		fmt.Println("Metrics:")
		// Print target metric first
		if value, exists := result.Metrics[string(targetMetric)]; exists {
			fmt.Printf("  %s: %.4f\n", targetMetric, value)
		}
		
		// Print other metrics
		for name, value := range result.Metrics {
			if name != string(targetMetric) {
				fmt.Printf("  %s: %.4f\n", name, value)
			}
		}
		
		fmt.Println()
	}
}

// FormatParameterSet formats a parameter set as a string
func FormatParameterSet(params ParameterSet) string {
	result := "{"
	
	// Get sorted parameter names for consistent output
	names := make([]string, 0, len(params))
	for name := range params {
		names = append(names, name)
	}
	sort.Strings(names)
	
	for i, name := range names {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%s: %v", name, params[name])
	}
	
	result += "}"
	return result
}

// CreateParameterSet creates a parameter set from the provided values
func CreateParameterSet(values map[string]interface{}) ParameterSet {
	params := make(ParameterSet)
	for name, value := range values {
		params[name] = value
	}
	return params
}

// MergeResults combines multiple result sets into a single slice
func MergeResults(resultSets ...[]*Result) []*Result {
	totalSize := 0
	for _, set := range resultSets {
		totalSize += len(set)
	}
	
	merged := make([]*Result, 0, totalSize)
	for _, set := range resultSets {
		merged = append(merged, set...)
	}
	
	return merged
}
