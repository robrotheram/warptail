package auth

import (
	"context"
	"warptail/pkg/utils"

	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type Users struct {
	db *bun.DB
}

func NewUserStore(db *bun.DB) *Users {
	store := Users{
		db: db,
	}
	store.CreateAdminUser()
	return &store
}

type Role string

const ADMIN = Role("admin")
const USER = Role("user")

type User struct {
	ID            uuid.UUID `bun:"id,type:uuid,pk,notnull" json:"id"`
	Name          string    `bun:",notnull" json:"name"`
	Email         string    `bun:",notnull" json:"email"`
	Password      string    `bun:",notnull" json:"password,omitempty"`
	Type          string    `bun:",notnull" json:"type"`
	Role          Role      `bun:",notnull" json:"role"`
	CreatedAt     time.Time `bun:",default:current_timestamp" json:"created_at"`
	bun.BaseModel `bun:"table:users,alias:u"`
}

func NewUser() User {
	return User{
		ID:   uuid.New(),
		Type: "internal",
	}
}

func (user *User) HashPassword() error {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedBytes)
	return nil
}

func (user *User) Sanatize() User {
	return User{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Type:      user.Type,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
	}
}

// CheckPassword compares a hashed password with a plain-text password.
func (user *User) VerifyPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(plainPassword))
	return err == nil
}

func (store *Users) Update(user User, id string, ctx context.Context) error {
	var existingUser User
	if err := store.db.NewSelect().Model(&existingUser).Where("id = ?", id).Scan(ctx); err != nil {
		return err
	}
	existingUser.Email = user.Email
	existingUser.Role = user.Role

	if len(user.Password) > 0 {
		existingUser.Password = user.Password
		existingUser.HashPassword()
	}

	_, err := store.db.NewUpdate().Model(&existingUser).Where("id = ?", id).Exec(ctx)

	return err
}

func (store *Users) List(ctx context.Context) ([]User, error) {
	var users []User
	err := store.db.NewSelect().Model(&users).Scan(ctx)
	return users, err
}

func (store *Users) Create(user User, ctx context.Context) error {
	if err := user.HashPassword(); err != nil {
		return err
	}
	if _, err := store.db.NewInsert().Model(&user).Exec(ctx); err != nil {
		return err
	}
	return nil
}

func (store *Users) FindByEamil(email string, ctx context.Context) (User, error) {
	var user User
	err := store.db.NewSelect().Model(&user).Where("email = ?", email).Scan(context.Background())
	return user, err
}

func (store *Users) FindByID(id string, ctx context.Context) (User, error) {
	var user User
	err := store.db.NewSelect().Model(&user).Where("id = ?", id).Scan(context.Background())
	return user, err
}

func (store *Users) CreateAdminUser() error {
	password := generatePassword(20, 2, 2, 2)
	admin := User{
		ID:       uuid.New(),
		Name:     "admin",
		Email:    "admin@warptail.local",
		Role:     ADMIN,
		Password: password,
	}

	_, err := store.FindByEamil(admin.Email, context.Background())
	if err == nil {
		return nil
	}
	utils.Logger.Info("New admin user created password", "password", password)
	admin.HashPassword()
	_, err = store.db.NewInsert().Model(&admin).Exec(context.Background())
	return err
}

func (store *Users) Delete(id string, ctx context.Context) error {
	_, err := store.db.NewDelete().Model(&User{}).Where("id = ?", id).Exec(ctx)
	return err
}
