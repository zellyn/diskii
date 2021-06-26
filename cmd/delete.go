// Copyright © 2016 Zellyn Hunter <zellyn@gmail.com>

package cmd

/*
var missingok bool // flag for whether to consider deleting a nonexistent file okay

// deleteCmd represents the delete command, used to delete a file.
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "delete a file",
	Long: `Delete a file.

delete disk-image.dsk HELLO
`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDelete(args); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(-1)
		}
	},
}

func init() {
	RootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().BoolVarP(&missingok, "missingok", "f", false, "if true, don't consider deleting a nonexistent file an error")
}

// runDelete performs the actual delete logic.
func runDelete(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("delete expects a disk image filename, and a filename")
	}
	op, err := disk.Open(args[0])
	if err != nil {
		return err
	}
	deleted, err := op.Delete(args[1])
	if err != nil {
		return err
	}
	if !deleted && !missingok {
		return fmt.Errorf("file %q not found", args[1])
	}
	f, err := os.Create(args[0])
	if err != nil {
		return err
	}
	_, err = op.Write(f)
	if err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return nil
}
*/
