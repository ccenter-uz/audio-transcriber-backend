package repo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/mirjalilova/voice_transcribe/config"
	"github.com/mirjalilova/voice_transcribe/internal/entity"
	"github.com/mirjalilova/voice_transcribe/pkg/logger"
	"github.com/mirjalilova/voice_transcribe/pkg/postgres"
)

type AuthRepo struct {
	pg     *postgres.Postgres
	config *config.Config
	logger *logger.Logger
}

// New -.
func NewAuthRepo(pg *postgres.Postgres, config *config.Config, logger *logger.Logger) *AuthRepo {
	return &AuthRepo{
		pg:     pg,
		config: config,
		logger: logger,
	}
}

func (r *AuthRepo) Login(ctx context.Context, req *entity.LoginReq) (*entity.UserInfo, error) {
	res := &entity.UserInfo{}

	var password string
	var createdAt time.Time
	query := `SELECT id, login, role, password_hash, service_name, username, first_number, image_url, created_at FROM users WHERE login = $1 AND password_hash = $2 AND deleted_at = 0`
	err := r.pg.Pool.QueryRow(ctx, query, req.Login, req.Password).Scan(
		&res.AgentID,
		&res.Login,
		&res.Role,
		&password,
		&res.ServiceName,
		&res.Name,
		&res.FirstNumber,
		&res.Image,
		&createdAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// if err := bcrypt.CompareHashAndPassword([]byte(password), []byte(req.Password)); err != nil {
	// 	return nil, errors.New("invalid login or password")
	// }

	res.CreateDate = createdAt.Format("2006-01-02 15:04:05")

	return res, nil
}

func (r *AuthRepo) Create(ctx context.Context, req *entity.UserInfo) error {

	query := `
	INSERT INTO users (id, login, username, password_hash, first_number, service_name, image_url)
	VALUES ($1, $2, $3, $4, $5, $6, $7) 
`
	_, err := r.pg.Pool.Exec(ctx, query, req.AgentID, req.Login, req.Name, req.Password, req.FirstNumber, req.ServiceName, req.Image)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

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
