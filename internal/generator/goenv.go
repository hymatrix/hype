package generator

func runGoInitAndTidy(projectDir string, module string) error {
	if err := runCmd(projectDir, "go", "mod", "init", module); err != nil {
		return err
	}
	if err := runCmd(projectDir, "go", "mod", "tidy"); err != nil {
		return err
	}
	return nil
}
