package sqlcrepo

import (
	"context"
	"valancis-backend/db/sqlc"
	"valancis-backend/internal/domain"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type userRepository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewUserRepository(db *pgxpool.Pool) domain.UserRepository {
	return &userRepository{
		db:      db,
		queries: sqlc.New(db),
	}
}

// --- Helpers ---

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return formatUUID(u.Bytes)
}

func formatUUID(b [16]byte) string {
	return formatHex(b[0:4]) + "-" + formatHex(b[4:6]) + "-" + formatHex(b[6:8]) + "-" + formatHex(b[8:10]) + "-" + formatHex(b[10:16])
}

func formatHex(b []byte) string {
	const hex = "0123456789abcdef"
	result := make([]byte, len(b)*2)
	for i, v := range b {
		result[i*2] = hex[v>>4]
		result[i*2+1] = hex[v&0x0f]
	}
	return string(result)
}

func stringToUUID(s string) pgtype.UUID {
	var u pgtype.UUID
	if s == "" {
		return u
	}
	u.Scan(s)
	return u
}

func pgtimeToTime(t pgtype.Timestamp) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --- Mappers ---

func sqlcUserToDomain(u sqlc.User) *domain.User {
	return &domain.User{
		ID:        uuidToString(u.ID),
		Email:     u.Email,
		Role:      u.Role,
		FirstName: ptrString(u.FirstName),
		LastName:  ptrString(u.LastName),
		Phone:     ptrString(u.Phone),
		Avatar:    ptrString(u.Avatar),
		CreatedAt: pgtimeToTime(u.CreatedAt),
		UpdatedAt: pgtimeToTime(u.UpdatedAt),
	}
}

func sqlcAddressToDomain(a sqlc.Address) domain.Address {
	return domain.Address{
		ID:           uuidToString(a.ID),
		UserID:       uuidToString(a.UserID),
		Label:        ptrString(a.Label),
		ContactEmail: ptrString(a.ContactEmail),
		Phone:        ptrString(a.Phone),
		FirstName:    ptrString(a.FirstName),
		LastName:     ptrString(a.LastName),
		DeliveryZone: ptrString(a.DeliveryZone),
		Division:     ptrString(a.Division),
		District:     ptrString(a.District),
		Thana:        ptrString(a.Thana),
		AddressLine:  ptrString(a.AddressLine),
		Landmark:     ptrString(a.Landmark),
		PostalCode:   ptrString(a.PostalCode),
		IsDefault:    a.IsDefault,
		CreatedAt:    pgtimeToTime(a.CreatedAt),
	}
}

func sqlcRefreshTokenToDomain(rt sqlc.RefreshToken) *domain.RefreshToken {
	return &domain.RefreshToken{
		Token:     rt.Token,
		UserID:    uuidToString(rt.UserID),
		ExpiresAt: pgtimeToTime(rt.ExpiresAt),
		CreatedAt: pgtimeToTime(rt.CreatedAt),
		Revoked:   rt.Revoked,
		Device:    ptrString(rt.Device),
	}
}

// --- Repository Implementation ---

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	created, err := r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		Email:     user.Email,
		Role:      user.Role,
		FirstName: strPtr(user.FirstName),
		LastName:  strPtr(user.LastName),
		Avatar:    strPtr(user.Avatar),
	})
	if err != nil {
		return err
	}
	user.ID = uuidToString(created.ID)
	user.CreatedAt = pgtimeToTime(created.CreatedAt)
	user.UpdatedAt = pgtimeToTime(created.UpdatedAt)
	return nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return sqlcUserToDomain(u), nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	u, err := r.queries.GetUserByID(ctx, stringToUUID(id))
	if err != nil {
		return nil, err
	}
	return sqlcUserToDomain(u), nil
}

func (r *userRepository) GetAll(ctx context.Context, limit, offset int) ([]*domain.User, int64, error) {
	users, err := r.queries.ListUsers(ctx, sqlc.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountUsers(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*domain.User, len(users))
	for i, u := range users {
		result[i] = sqlcUserToDomain(u)
	}
	return result, count, nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	_, err := r.queries.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:        stringToUUID(user.ID),
		Email:     user.Email,
		Role:      user.Role,
		FirstName: strPtr(user.FirstName),
		LastName:  strPtr(user.LastName),
		Avatar:    strPtr(user.Avatar),
	})
	return err
}

func (r *userRepository) UpdateProfile(ctx context.Context, id, firstName, lastName, phone string) (*domain.User, error) {
	u, err := r.queries.UpdateUserProfile(ctx, sqlc.UpdateUserProfileParams{
		ID:        stringToUUID(id),
		FirstName: strPtr(firstName),
		LastName:  strPtr(lastName),
		Phone:     strPtr(phone),
	})
	if err != nil {
		return nil, err
	}
	return sqlcUserToDomain(u), nil
}

// --- Addresses ---

func (r *userRepository) AddAddress(ctx context.Context, addr *domain.Address) error {
	created, err := r.queries.CreateAddress(ctx, sqlc.CreateAddressParams{
		UserID:       stringToUUID(addr.UserID),
		Label:        strPtr(addr.Label),
		ContactEmail: strPtr(addr.ContactEmail),
		Phone:        strPtr(addr.Phone),
		FirstName:    strPtr(addr.FirstName),
		LastName:     strPtr(addr.LastName),
		DeliveryZone: strPtr(addr.DeliveryZone),
		Division:     strPtr(addr.Division),
		District:     strPtr(addr.District),
		Thana:        strPtr(addr.Thana),
		AddressLine:  strPtr(addr.AddressLine),
		Landmark:     strPtr(addr.Landmark),
		PostalCode:   strPtr(addr.PostalCode),
		IsDefault:    addr.IsDefault,
	})
	if err != nil {
		return err
	}
	addr.ID = uuidToString(created.ID)
	addr.CreatedAt = pgtimeToTime(created.CreatedAt)
	return nil
}

func (r *userRepository) UpdateAddress(ctx context.Context, addr *domain.Address) error {
	updated, err := r.queries.UpdateAddress(ctx, sqlc.UpdateAddressParams{
		ID:           stringToUUID(addr.ID),
		UserID:       stringToUUID(addr.UserID),
		Label:        strPtr(addr.Label),
		ContactEmail: strPtr(addr.ContactEmail),
		Phone:        strPtr(addr.Phone),
		FirstName:    strPtr(addr.FirstName),
		LastName:     strPtr(addr.LastName),
		DeliveryZone: strPtr(addr.DeliveryZone),
		Division:     strPtr(addr.Division),
		District:     strPtr(addr.District),
		Thana:        strPtr(addr.Thana),
		AddressLine:  strPtr(addr.AddressLine),
		Landmark:     strPtr(addr.Landmark),
		PostalCode:   strPtr(addr.PostalCode),
		IsDefault:    addr.IsDefault,
	})
	if err != nil {
		return err
	}
	addr.CreatedAt = pgtimeToTime(updated.CreatedAt)
	return nil
}

func (r *userRepository) GetAddresses(ctx context.Context, userID string) ([]domain.Address, error) {
	addrs, err := r.queries.GetAddressesByUserID(ctx, stringToUUID(userID))
	if err != nil {
		return nil, err
	}
	result := make([]domain.Address, len(addrs))
	for i, a := range addrs {
		result[i] = sqlcAddressToDomain(a)
	}
	return result, nil
}

// --- Refresh Tokens ---

func (r *userRepository) SaveRefreshToken(ctx context.Context, token *domain.RefreshToken) error {
	var expiresAt pgtype.Timestamp
	expiresAt.Time = token.ExpiresAt
	expiresAt.Valid = true

	_, err := r.queries.SaveRefreshToken(ctx, sqlc.SaveRefreshTokenParams{
		Token:     token.Token,
		UserID:    stringToUUID(token.UserID),
		ExpiresAt: expiresAt,
		Device:    strPtr(token.Device),
	})
	return err
}

func (r *userRepository) GetRefreshToken(ctx context.Context, token string) (*domain.RefreshToken, error) {
	rt, err := r.queries.GetRefreshToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return sqlcRefreshTokenToDomain(rt), nil
}

func (r *userRepository) RevokeRefreshToken(ctx context.Context, token string) error {
	return r.queries.RevokeRefreshToken(ctx, token)
}

func (r *userRepository) DeleteAddress(ctx context.Context, id, userID string) error {
	return r.queries.DeleteAddress(ctx, sqlc.DeleteAddressParams{
		ID:     stringToUUID(id),
		UserID: stringToUUID(userID),
	})
}
