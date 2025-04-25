package repo

// import (
// 	"context"
// 	"database/sql"
// 	"errors"
// 	"fmt"
// 	"strconv"
// 	"strings"
// 	"time"

// 	"github.com/mirjalilova/voice_transcribe/config"
// 	"github.com/mirjalilova/voice_transcribe/internal/entity"
// 	"github.com/mirjalilova/voice_transcribe/pkg/logger"
// 	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
// 	"golang.org/x/crypto/bcrypt"
// )

// type AuthRepo struct {
// 	pg     *postgres.Postgres
// 	config *config.Config
// 	logger *logger.Logger
// }

// // New -.
// func NewAuthRepo(pg *postgres.Postgres, config *config.Config, logger *logger.Logger) *AuthRepo {
// 	return &AuthRepo{
// 		pg:     pg,
// 		config: config,
// 		logger: logger,
// 	}
// }

// func (r *AuthRepo) Login(ctx context.Context, req *entity.LoginReq) (*entity.User, error) {
// 	res := &entity.User{}

// 	var password string
// 	var createdAt time.Time
// 	query := `SELECT id, username, role, password_hash, created_at FROM users WHERE username = $1 AND deleted_at = 0`
// 	err := r.pg.Pool.QueryRow(ctx, query, req.Username).Scan(
// 		&res.Id,
// 		&res.Username,
// 		&res.Role,
// 		&password,
// 		&createdAt)
// 	if err != nil {
// 		if err == sql.ErrNoRows {
// 			return nil, fmt.Errorf("user not found: %w", err)
// 		}
// 		return nil, fmt.Errorf("failed to query user: %w", err)
// 	}

// 	if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
// 		return nil, errors.New("invalid username or password")
// 	}

// 	res.CreatedAt = createdAt.Format("2006-01-02 15:04:05")

// 	return res, nil
// }

// func (r *AuthRepo) Create(ctx context.Context, req *entity.CreateUser) error {

// 	query := `
// 	INSERT INTO users (username, password_hash, role)
// 	VALUES ($1, $2, $3) 
// `
// 	_, err := r.pg.Pool.Exec(ctx, query, req.Username, req.Password, req.Role)
// 	if err != nil {
// 		return fmt.Errorf("failed to create user: %w", err)
// 	}

// 	return nil
// }

// func (r *AuthRepo) Update(ctx context.Context, req *entity.UpdateUser) error {
// 	query := `
// 	UPDATE
// 		users
// 	SET`

// 	var conditions []string
// 	var args []interface{}

// 	if req.Username != "" && req.Username != "string" {
// 		conditions = append(conditions, " username = $"+strconv.Itoa(len(args)+1))
// 		args = append(args, req.Username)
// 	}
// 	if req.Role != "" && req.Role != "string" {
// 		conditions = append(conditions, " role = $"+strconv.Itoa(len(args)+1))
// 		args = append(args, req.Role)
// 	}

// 	if len(conditions) == 0 {
// 		return errors.New("nothing to update")
// 	}

// 	conditions = append(conditions, " updated_at = now()")
// 	query += strings.Join(conditions, ", ")
// 	query += " WHERE id = $" + strconv.Itoa(len(args)+1) + " AND deleted_at = 0"

// 	args = append(args, req.Id)

// 	_, err := r.pg.Pool.Exec(ctx, query, args...)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (r *AuthRepo) GetById(ctx context.Context, id int) (*entity.User, error) {
// 	var createdAt time.Time

// 	query := `
// 	SELECT id, username, role, created_at
// 	FROM users
// 	WHERE id = $1 AND deleted_at = 0
// 	`
// 	user := &entity.User{}
// 	err := r.pg.Pool.QueryRow(ctx, query, id).Scan(&user.Id, &user.Username, &user.Role, &createdAt)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user: %w", err)
// 	}

// 	user.CreatedAt = createdAt.Format("2006-01-02 15:04:05")

// 	return user, nil
// }

// func (r *AuthRepo) GetList(ctx context.Context, req *entity.GetUserReq) (*entity.UserList, error) {
// 	query := `
// 	SELECT COUNT(id) OVER () AS total_count, id, username, role, created_at
// 	FROM users
// 	WHERE deleted_at = 0
// 	`

// 	var conditions []string
// 	var args []interface{}
// 	if req.Username != "" {
// 		conditions = append(conditions, " username ILIKE $"+strconv.Itoa(len(args)+1))
// 		args = append(args, "%"+req.Username+"%")
// 	}
// 	if req.Role != "" {
// 		conditions = append(conditions, " role = $"+strconv.Itoa(len(args)+1))
// 		args = append(args, req.Role)
// 	}

// 	if len(conditions) > 0 {
// 		query += " AND " + strings.Join(conditions, " AND ")
// 	}

// 	query += ` ORDER BY created_at DESC OFFSET $` + strconv.Itoa(len(args)+1) + ` LIMIT $` + strconv.Itoa(len(args)+2)

// 	args = append(args, req.Filter.Offset, req.Filter.Limit)

// 	rows, err := r.pg.Pool.Query(ctx, query, args...)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get user list: %w", err)
// 	}
// 	defer rows.Close()

// 	users := entity.UserList{}
// 	for rows.Next() {
// 		var count int
// 		var createdAt time.Time
// 		user := entity.User{}
// 		err := rows.Scan(&count, &user.Id, &user.Username, &user.Role, &createdAt)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to scan user: %w", err)
// 		}
// 		user.CreatedAt = createdAt.Format("2006-01-02 15:04:05")
// 		users.Users = append(users.Users, user)
// 		users.Count = count
// 	}

// 	return &users, nil
// }

// func (r *AuthRepo) Delete(ctx context.Context, id int) error {
// 	query := `
// 	UPDATE users
// 	SET deleted_at = EXTRACT(EPOCH FROM NOW())
// 	WHERE id = $1 AND deleted_at = 0
// 	`
// 	_, err := r.pg.Pool.Exec(ctx, query, id)
// 	if err != nil {
// 		return fmt.Errorf("failed to delete user: %w", err)
// 	}

// 	return nil
// }
