package main

import "embed"

//go:embed www/*
var webUI embed.FS
