#!/bin/bash

# Script to download historical data for backtesting the ChatGPT strategy
# Usage: ./download_data.sh [symbol] [timeframe] [days]

# Default values
SYMBOL=${1:-"BTCUSDT"}
TIMEFRAME=${2:-"5m"}
DAYS=${3:-30}

# Create data directory if it doesn't exist
mkdir -p data

# Calculate start and end dates
END_DATE=$(date +%Y-%m-%d)
START_DATE=$(date -d "$END_DATE -$DAYS days" +%Y-%m-%d)

echo "Downloading $SYMBOL data for $TIMEFRAME timeframe from $START_DATE to $END_DATE"

# Use Binance API to download data
# This is a simplified version - in a real implementation, you would need to handle pagination and rate limits
curl -s "https://api.binance.com/api/v3/klines?symbol=$SYMBOL&interval=$TIMEFRAME&startTime=$(date -d "$START_DATE" +%s)000&endTime=$(date -d "$END_DATE" +%s)000&limit=1000" | \
jq -r '.[] | [.[0]/1000|strftime("%Y-%m-%d %H:%M:%S"), .[1], .[2], .[3], .[4], .[5]] | @csv' > "data/$SYMBOL-$TIMEFRAME.csv"

# Check if download was successful
if [ -s "data/$SYMBOL-$TIMEFRAME.csv" ]; then
    echo "Data downloaded successfully to data/$SYMBOL-$TIMEFRAME.csv"
    echo "Number of candles: $(wc -l < "data/$SYMBOL-$TIMEFRAME.csv")"
else
    echo "Error: Failed to download data"
    exit 1
fi

# Format the CSV file to match the expected format for the CSV feed
# The expected format is: timestamp,open,high,low,close,volume
echo "Formatting CSV file..."
mv "data/$SYMBOL-$TIMEFRAME.csv" "data/$SYMBOL-$TIMEFRAME.tmp"
echo "timestamp,open,high,low,close,volume" > "data/$SYMBOL-$TIMEFRAME.csv"
cat "data/$SYMBOL-$TIMEFRAME.tmp" >> "data/$SYMBOL-$TIMEFRAME.csv"
rm "data/$SYMBOL-$TIMEFRAME.tmp"

echo "Data preparation complete"
