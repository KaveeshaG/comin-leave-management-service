-- migrations/000001_create_initial_schema.up.sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Leave Types (e.g., Vacation, Sick, Personal)
CREATE TABLE leave_types (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    color VARCHAR(7),  -- For UI representation
    default_days INTEGER,
    is_paid BOOLEAN DEFAULT true,
    requires_approval BOOLEAN DEFAULT true,
    min_days_notice INTEGER DEFAULT 0,
    max_days_per_request INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, name)
);

-- Leave Balances
CREATE TABLE leave_balances (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL,
    employee_id UUID NOT NULL,
    leave_type_id UUID REFERENCES leave_types(id),
    year INTEGER NOT NULL,
    total_days DECIMAL(5,2) NOT NULL,
    used_days DECIMAL(5,2) DEFAULT 0,
    pending_days DECIMAL(5,2) DEFAULT 0,
    remaining_days DECIMAL(5,2) GENERATED ALWAYS AS (total_days - used_days - pending_days) STORED,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(employee_id, leave_type_id, year)
);

-- Leave Requests
CREATE TABLE leave_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL,
    employee_id UUID NOT NULL,
    leave_type_id UUID REFERENCES leave_types(id),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    days DECIMAL(5,2) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected, cancelled
    reason TEXT,
    comments TEXT,
    approved_by UUID,
    approved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Leave Request History
CREATE TABLE leave_request_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    leave_request_id UUID REFERENCES leave_requests(id),
    action VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    comments TEXT,
    performed_by UUID NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Holidays
CREATE TABLE holidays (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    organization_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    date DATE NOT NULL,
    type VARCHAR(20) NOT NULL, -- public, company, optional
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(organization_id, date)
);

CREATE TABLE leave_balance_adjustments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    leave_balance_id UUID NOT NULL REFERENCES leave_balances(id),
    adjustment DECIMAL(5,2) NOT NULL,
    reason TEXT NOT NULL,
    performed_by UUID NOT NULL,
    approved_by UUID,
    approved_at TIMESTAMP WITH TIME ZONE,
    comments TEXT,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE leave_types ADD CONSTRAINT unique_org_leave_type_name UNIQUE (organization_id, name);

-- Create indexes
CREATE INDEX idx_leave_types_org ON leave_types(organization_id);
CREATE INDEX idx_leave_balances_employee ON leave_balances(employee_id);
CREATE INDEX idx_leave_balances_type ON leave_balances(leave_type_id);
CREATE INDEX idx_leave_requests_employee ON leave_requests(employee_id);
CREATE INDEX idx_leave_requests_type ON leave_requests(leave_type_id);
CREATE INDEX idx_leave_requests_dates ON leave_requests(start_date, end_date);
CREATE INDEX idx_holidays_org_date ON holidays(organization_id, date);
CREATE INDEX idx_leave_balance_adjustments_balance ON leave_balance_adjustments(leave_balance_id);