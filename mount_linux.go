package main

func unmountArgs(base string) []string {
	return []string{"fusermount", "-u", base}
}
