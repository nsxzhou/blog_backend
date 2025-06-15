package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nsxzhou1114/blog-api/internal/database"
	"github.com/nsxzhou1114/blog-api/internal/model"
	"github.com/nsxzhou1114/blog-api/internal/service"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

// userCmd 用户管理命令
var userCmd = &cobra.Command{
	Use:   "user",
	Short: "用户管理命令",
	Long:  `用户管理相关的命令，包括创建管理员、列出用户、重置密码等`,
}

// createAdminCmd 创建管理员用户命令
var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "创建管理员用户",
	Long:  `交互式创建管理员用户`,
	Run: func(cmd *cobra.Command, args []string) {
		createAdminUser()
	},
}

// listUsersCmd 列出用户命令
var listUsersCmd = &cobra.Command{
	Use:   "list",
	Short: "列出用户",
	Long:  `列出系统中的用户`,
	Run: func(cmd *cobra.Command, args []string) {
		listUsers()
	},
}

// resetPasswordCmd 重置用户密码命令
var resetPasswordCmd = &cobra.Command{
	Use:   "reset-password [username]",
	Short: "重置用户密码",
	Long:  `重置指定用户的密码`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		resetUserPassword(args[0])
	},
}

// updateUserStatusCmd 更新用户状态命令
var updateUserStatusCmd = &cobra.Command{
	Use:   "update-status [username] [status]",
	Short: "更新用户状态",
	Long:  `更新用户状态 (0=禁用, 1=启用)`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		updateUserStatus(args[0], args[1])
	},
}

func init() {
	// 添加用户相关子命令
	userCmd.AddCommand(createAdminCmd)
	userCmd.AddCommand(listUsersCmd)
	userCmd.AddCommand(resetPasswordCmd)
	userCmd.AddCommand(updateUserStatusCmd)

	// 将用户命令添加到根命令
	rootCmd.AddCommand(userCmd)
}

// createAdminUser 创建管理员用户
func createAdminUser() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Print("请输入管理员用户名: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("请输入管理员邮箱: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	fmt.Print("请输入管理员密码: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("读取密码失败: %v\n", err)
		return
	}
	password := string(passwordBytes)
	fmt.Println() // 换行

	fmt.Print("请确认管理员密码: ")
	confirmPasswordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("读取确认密码失败: %v\n", err)
		return
	}
	confirmPassword := string(confirmPasswordBytes)
	fmt.Println() // 换行

	if password != confirmPassword {
		fmt.Println("两次输入的密码不一致")
		return
	}

	// 密码加密
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		fmt.Printf("密码加密失败: %v\n", err)
		return
	}

	userService := service.NewUserService()

	// 检查用户名是否已存在
	if _, err := userService.GetUserByUsername(username); err == nil {
		fmt.Println("用户名已存在")
		return
	}

	// 创建管理员用户
	user := &model.User{
		Username:        username,
		Password:        string(hashedPassword),
		Email:           email,
		Role:            "admin",
		Status:          1,
		IsEmailVerified: 1,
		IsPhoneVerified: 1,
		LastLoginAt:     time.Now(),
		LastLoginIP:     "127.0.0.1",
	}

	db := database.GetDB()
	if err := db.Create(user).Error; err != nil {
		fmt.Printf("创建管理员用户失败: %v\n", err)
		return
	}

	fmt.Printf("管理员用户创建成功！\n")
	fmt.Printf("用户名: %s\n", username)
	fmt.Printf("邮箱: %s\n", email)
}

// listUsers 列出用户
func listUsers() {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	db := database.GetDB()
	var users []model.User

	if err := db.Select("id, username, email, role, status, created_at, last_login_at").
		Order("created_at DESC").
		Limit(50).
		Find(&users).Error; err != nil {
		fmt.Printf("查询用户列表失败: %v\n", err)
		return
	}

	fmt.Printf("%-5s %-20s %-30s %-20s %-10s %-8s %-20s\n",
		"ID", "用户名", "邮箱", "角色", "状态", "创建时间", "最后登录")
	fmt.Println(strings.Repeat("-", 100))

	for _, user := range users {
		status := "启用"
		if user.Status == 0 {
			status = "禁用"
		}

		lastLogin := "从未登录"
		if !user.LastLoginAt.IsZero() {
			lastLogin = user.LastLoginAt.Format("2006-01-02 15:04")
		}

		fmt.Printf("%-5d %-20s %-30s %-20s %-10s %-8s %-20s\n",
			user.ID, user.Username, user.Email,
			user.Role, status, user.CreatedAt.Format("2006-01-02 15:04"), lastLogin)
	}
}

// resetUserPassword 重置用户密码
func resetUserPassword(username string) {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	userService := service.NewUserService()
	user, err := userService.GetUserByUsername(username)
	if err != nil {
		fmt.Printf("用户不存在: %v\n", err)
		return
	}

	fmt.Print("请输入新密码: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("读取密码失败: %v\n", err)
		return
	}
	password := string(passwordBytes)
	fmt.Println() // 换行

	fmt.Print("请确认新密码: ")
	confirmPasswordBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		fmt.Printf("读取确认密码失败: %v\n", err)
		return
	}
	confirmPassword := string(confirmPasswordBytes)
	fmt.Println() // 换行

	if password != confirmPassword {
		fmt.Println("两次输入的密码不一致")
		return
	}

	// 重置密码
	if err := userService.ResetUserPassword(user.ID, password); err != nil {
		fmt.Printf("重置密码失败: %v\n", err)
		return
	}

	fmt.Printf("用户 %s 的密码重置成功！\n", username)
}

// updateUserStatus 更新用户状态
func updateUserStatus(username, statusStr string) {
	if err := initializeSystem(); err != nil {
		fmt.Printf("系统初始化失败: %v\n", err)
		os.Exit(1)
	}

	status, err := strconv.Atoi(statusStr)
	if err != nil || (status != 0 && status != 1) {
		fmt.Println("状态值必须是 0 (禁用) 或 1 (启用)")
		return
	}

	userService := service.NewUserService()
	user, err := userService.GetUserByUsername(username)
	if err != nil {
		fmt.Printf("用户不存在: %v\n", err)
		return
	}

	if err := userService.UpdateUserStatus(user.ID, status); err != nil {
		fmt.Printf("更新用户状态失败: %v\n", err)
		return
	}

	statusText := "启用"
	if status == 0 {
		statusText = "禁用"
	}

	fmt.Printf("用户 %s 的状态已更新为: %s\n", username, statusText)
}
