-- +goose Up

-- Перечисление ролей участника организации
CREATE TYPE membership_role AS ENUM ('OWNER', 'ADMIN', 'MEMBER', 'VIEWER');

-- Перечисление статусов приглашения
CREATE TYPE invite_status AS ENUM ('PENDING', 'ACCEPTED', 'DECLINED', 'EXPIRED', 'REVOKED');

-- Организация владеет всеми ресурсами и подпиской
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    tier VARCHAR(50) NOT NULL DEFAULT 'Free',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_organizations_owner_id ON organizations(owner_id);
CREATE INDEX idx_organizations_tier ON organizations(tier);

-- Связь many-to-many пользователей и организаций с ролью
CREATE TABLE memberships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role membership_role NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_active_at TIMESTAMPTZ,
    UNIQUE (organization_id, user_id)
);

-- В каждой организации обязан быть ровно один OWNER. Уникальный частичный
-- индекс гарантирует это правило на уровне БД.
CREATE UNIQUE INDEX idx_memberships_one_owner
    ON memberships(organization_id)
    WHERE role = 'OWNER';

CREATE INDEX idx_memberships_user_id ON memberships(user_id);
CREATE INDEX idx_memberships_organization_id ON memberships(organization_id);

-- Приглашения нового участника в организацию
CREATE TABLE invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    inviter_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    email VARCHAR(255) NOT NULL,
    role membership_role NOT NULL,
    token UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    status invite_status NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    CHECK (role <> 'OWNER')
);

-- Только одно PENDING приглашение на email в рамках организации
CREATE UNIQUE INDEX idx_invites_unique_pending
    ON invites(organization_id, email)
    WHERE status = 'PENDING';

CREATE INDEX idx_invites_organization_id ON invites(organization_id);
CREATE INDEX idx_invites_email ON invites(email);
CREATE INDEX idx_invites_token ON invites(token);
CREATE INDEX idx_invites_status ON invites(status);
CREATE INDEX idx_invites_expires_at ON invites(expires_at) WHERE status = 'PENDING';

-- +goose Down
DROP INDEX IF EXISTS idx_invites_expires_at;
DROP INDEX IF EXISTS idx_invites_status;
DROP INDEX IF EXISTS idx_invites_token;
DROP INDEX IF EXISTS idx_invites_email;
DROP INDEX IF EXISTS idx_invites_organization_id;
DROP INDEX IF EXISTS idx_invites_unique_pending;
DROP TABLE IF EXISTS invites;

DROP INDEX IF EXISTS idx_memberships_organization_id;
DROP INDEX IF EXISTS idx_memberships_user_id;
DROP INDEX IF EXISTS idx_memberships_one_owner;
DROP TABLE IF EXISTS memberships;

DROP INDEX IF EXISTS idx_organizations_tier;
DROP INDEX IF EXISTS idx_organizations_owner_id;
DROP TABLE IF EXISTS organizations;

DROP TYPE IF EXISTS invite_status;
DROP TYPE IF EXISTS membership_role;
