package core

import "crypto/rand"

func NewID(prefix string) string { return prefix + "_" + rand.Text() }
