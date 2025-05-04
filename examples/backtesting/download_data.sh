#!/bin/bash

echo "Selecione o tipo de comando:"
echo "1) Período entre datas (-s e -e)"
echo "2) Últimos N dias (--days)"
read -p "Opção [1/2]: " option

read -p "Pares de moedas separados por espaço (ex: BTCUSDT ETHUSDT): " pairs
read -p "Timeframe (ex: 1h, 1d): " timeframe

if [ "$option" == "1" ]; then
    read -p "Data de início (ex: 2020-12-01): " start
    read -p "Data de fim (ex: 2020-12-31): " end
    read -p "Pasta de saída (ex: ./dados): " output_dir

    mkdir -p "$output_dir"  # cria a pasta se não existir

    for pair in $pairs; do
        filename="${pair}-${timeframe}.csv"
        output="${output_dir}/${filename}"
        echo "Baixando $pair..."
        go run ./../../cmd/backnrun/main.go download -p "$pair" -t "$timeframe" -s "$start" -e "$end" -o "$output"
    done

elif [ "$option" == "2" ]; then
    read -p "Número de dias: " days
    read -p "Pasta de saída (ex: ./dados): " output_dir

    mkdir -p "$output_dir"

    for pair in $pairs; do
        filename="${pair}-${timeframe}.csv"
        output="${output_dir}/${filename}"
        echo "Baixando $pair..."
        go run ./../../cmd/backnrun/main.go download --pair "$pair" --timeframe "$timeframe" --days "$days" --output "$output"
    done

else
    echo "Opção inválida!"
    exit 1
fi
