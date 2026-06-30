-- name: GetUserCompanyIDs :many
SELECT company_id
FROM user_companies
WHERE user_id = $1
ORDER BY company_id;
