package user

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/kofuk/premises/controlpanel/internal/db"
	"github.com/kofuk/premises/controlpanel/internal/db/model"
	"github.com/spf13/cobra"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

func NewUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:  "user",
		Long: "User-related functionality.",
	}
	cmd.AddCommand(NewAddCommand())
	cmd.AddCommand(NewResetPasswordCommand())
	cmd.AddCommand(NewRenameCommand())

	return cmd
}

type AddUserOptions struct {
	Name          string
	Password      string
	PasswordStdin bool
	Initialized   bool
}

type ResetPasswordOptions struct {
	Name          string
	Password      string
	PasswordStdin bool
}

type RenameOptions struct {
	Name    string
	NewName string
}

func NewAddCommand() *cobra.Command {
	var options AddUserOptions

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunAddUser(options)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&options.Name, "username", "u", "", "Username")
	flags.StringVarP(&options.Password, "password", "p", "", "Password")
	flags.BoolVar(&options.PasswordStdin, "password-stdin", false, "Read password from stdin")
	flags.BoolVar(&options.Initialized, "initialized", false, "Mark this user as initialized")

	return cmd
}

func NewResetPasswordCommand() *cobra.Command {
	var options ResetPasswordOptions

	cmd := &cobra.Command{
		Use:   "reset-password",
		Short: "Reset password for an existing user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunResetPassword(options)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&options.Name, "username", "u", "", "Username")
	flags.StringVarP(&options.Password, "password", "p", "", "Password")
	flags.BoolVar(&options.PasswordStdin, "password-stdin", false, "Read password from stdin")

	return cmd
}

func NewRenameCommand() *cobra.Command {
	var options RenameOptions

	cmd := &cobra.Command{
		Use:   "rename",
		Short: "Rename an existing user",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunRename(options)
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&options.Name, "username", "u", "", "Username")
	flags.StringVarP(&options.NewName, "new-name", "t", "", "New username")

	return cmd
}

func createClient() *bun.DB {
	addr := os.Getenv("PREMISES_POSTGRES_ADDRESS")
	user := os.Getenv("PREMISES_POSTGRES_USER")
	password := os.Getenv("PREMISES_POSTGRES_PASSWORD")
	database := os.Getenv("PREMISES_POSTGRES_DB")

	if addr == "" || user == "" || password == "" || database == "" {
		fmt.Fprintln(os.Stderr, "Database configuration is missing")
		os.Exit(1)
	}

	return db.NewClient(addr, user, password, database)
}

func readPasswordStdin() (string, error) {
	r := bufio.NewReader(os.Stdin)
	l, _, err := r.ReadLine()
	if err != nil {
		return "", err
	}
	return string(l), nil
}

func RunAddUser(options AddUserOptions) error {
	password := options.Password
	if options.PasswordStdin {
		var err error
		password, err = readPasswordStdin()
		if err != nil {
			return err
		}
	}

	db := createClient()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &model.User{
		Name:        options.Name,
		Password:    string(hashedPassword),
		Initialized: options.Initialized,
	}

	if _, err := db.NewInsert().Model(user).Exec(context.TODO()); err != nil {
		return err
	}

	return nil
}

func RunResetPassword(options ResetPasswordOptions) error {
	password := options.Password
	if options.PasswordStdin {
		var err error
		password, err = readPasswordStdin()
		if err != nil {
			return err
		}
	}

	db := createClient()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	if _, err := db.NewUpdate().Model((*model.User)(nil)).Set("password = ?", string(hashedPassword)).Set("initialized = ?", false).Where("name = ? AND deleted_at IS NULL", options.Name).Exec(context.TODO()); err != nil {
		return err
	}

	return nil
}

func RunRename(options RenameOptions) error {
	db := createClient()

	if _, err := db.NewUpdate().Model((*model.User)(nil)).Set("name = ?", options.NewName).Where("name = ? AND deleted_at IS NULL", options.Name).Exec(context.TODO()); err != nil {
		return err
	}

	return nil
}
