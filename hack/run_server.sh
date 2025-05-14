#! /bin/env bash

MODEL=${MODEL:-"gemma2"}

ollama_unit() {
    local action="$1"
    if [[ -z "$action" ]]; then
        return
    fi
    sudo systemctl "$action" ollama
}

run_llm() {
    local model="$1"
    if [[ -z "$model" ]]; then
        echo "No model provided"
        exit 1
    fi
    ollama run "$model"
}

ollama_unit "start"
