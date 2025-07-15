-- WindGo Chat Database Initialization Script
-- This script initializes the PostgreSQL database matching GORM models

-- Set timezone and encoding
SET timezone = 'UTC';
SET client_encoding = 'UTF8';

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "citext";

-- Insert initial data only (GORM will handle table creation)

-- Wait for GORM to create tables, then insert demo data
-- This will be run after GORM migration

-- Insert default rooms (only if table exists and is empty)
DO $$
BEGIN
    -- Check if rooms table exists and insert default rooms
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'rooms') THEN
        INSERT INTO rooms (name, created_at, updated_at)
        SELECT * FROM (VALUES
            ('General', NOW(), NOW()),
            ('Random', NOW(), NOW()),
            ('Tech Talk', NOW(), NOW()),
            ('Announcements', NOW(), NOW())
        ) AS new_rooms(name, created_at, updated_at)
        WHERE NOT EXISTS (SELECT 1 FROM rooms WHERE name = new_rooms.name);
    END IF;
END $$;

-- Insert demo users (only if table exists and is empty)
DO $$
BEGIN
    -- Check if users table exists and insert demo users
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users') THEN
        -- Create demo users with hashed passwords (password: admin123)
        -- Hash: $2a$10$1xCOKVytGk8KJ8r8rQQP4.GkNYyTMLxoLxMPkGSBPqJt2TzDYzr.C
        INSERT INTO users (username, email, password, created_at, updated_at)
        SELECT * FROM (VALUES
            ('admin', 'admin@windgo.com', '$2a$10$1xCOKVytGk8KJ8r8rQQP4.GkNYyTMLxoLxMPkGSBPqJt2TzDYzr.C', NOW(), NOW()),
            ('demo_user', 'demo@windgo.com', '$2a$10$1xCOKVytGk8KJ8r8rQQP4.GkNYyTMLxoLxMPkGSBPqJt2TzDYzr.C', NOW(), NOW())
        ) AS new_users(username, email, password, created_at, updated_at)
        WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = new_users.email);
    END IF;
END $$;

-- Insert welcome messages (only if tables exist)
DO $$
DECLARE
    admin_user_id INTEGER;
    general_room_id INTEGER;
    random_room_id INTEGER;
    tech_room_id INTEGER;
    announcements_room_id INTEGER;
BEGIN
    -- Check if all required tables exist
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users') AND
       EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'rooms') AND
       EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'messages') THEN

        -- Get admin user ID
        SELECT id INTO admin_user_id FROM users WHERE username = 'admin' LIMIT 1;

        -- Get room IDs
        SELECT id INTO general_room_id FROM rooms WHERE name = 'General' LIMIT 1;
        SELECT id INTO random_room_id FROM rooms WHERE name = 'Random' LIMIT 1;
        SELECT id INTO tech_room_id FROM rooms WHERE name = 'Tech Talk' LIMIT 1;
        SELECT id INTO announcements_room_id FROM rooms WHERE name = 'Announcements' LIMIT 1;

        -- Insert welcome messages if admin user and rooms exist
        IF admin_user_id IS NOT NULL THEN
            -- General room welcome message
            IF general_room_id IS NOT NULL THEN
                INSERT INTO messages (content, user_id, room_id, created_at, updated_at)
                SELECT 'Welcome to WindGo Chat! This is the general discussion room where everyone can chat.', admin_user_id, general_room_id, NOW(), NOW()
                WHERE NOT EXISTS (SELECT 1 FROM messages WHERE room_id = general_room_id);
            END IF;

            -- Random room welcome message
            IF random_room_id IS NOT NULL THEN
                INSERT INTO messages (content, user_id, room_id, created_at, updated_at)
                SELECT 'Feel free to chat about anything here! This is our random discussion room.', admin_user_id, random_room_id, NOW(), NOW()
                WHERE NOT EXISTS (SELECT 1 FROM messages WHERE room_id = random_room_id);
            END IF;

            -- Tech Talk room welcome message
            IF tech_room_id IS NOT NULL THEN
                INSERT INTO messages (content, user_id, room_id, created_at, updated_at)
                SELECT 'Share your tech knowledge and learn from others! Let''s discuss programming, frameworks, and development.', admin_user_id, tech_room_id, NOW(), NOW()
                WHERE NOT EXISTS (SELECT 1 FROM messages WHERE room_id = tech_room_id);
            END IF;

            -- Announcements room welcome message
            IF announcements_room_id IS NOT NULL THEN
                INSERT INTO messages (content, user_id, room_id, created_at, updated_at)
                SELECT 'This is where important announcements and updates will be posted. Stay tuned!', admin_user_id, announcements_room_id, NOW(), NOW()
                WHERE NOT EXISTS (SELECT 1 FROM messages WHERE room_id = announcements_room_id);
            END IF;
        END IF;
    END IF;
END $$;

-- Create a function to check database health
CREATE OR REPLACE FUNCTION check_db_health()
RETURNS TABLE(
    table_name TEXT,
    row_count BIGINT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 'users'::TEXT, COALESCE((SELECT COUNT(*) FROM users), 0)::BIGINT
    UNION ALL
    SELECT 'rooms'::TEXT, COALESCE((SELECT COUNT(*) FROM rooms), 0)::BIGINT
    UNION ALL
    SELECT 'messages'::TEXT, COALESCE((SELECT COUNT(*) FROM messages), 0)::BIGINT;
END;
$$ LANGUAGE plpgsql;

-- Grant all necessary permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;

-- Print initialization summary
DO $$
DECLARE
    user_count INTEGER := 0;
    room_count INTEGER := 0;
    message_count INTEGER := 0;
BEGIN
    -- Safely get counts
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'users') THEN
        SELECT COUNT(*) INTO user_count FROM users;
    END IF;

    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'rooms') THEN
        SELECT COUNT(*) INTO room_count FROM rooms;
    END IF;

    IF EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'messages') THEN
        SELECT COUNT(*) INTO message_count FROM messages;
    END IF;

    RAISE NOTICE '========================================';
    RAISE NOTICE 'WindGo Chat Database Initialized Successfully!';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Users created: %', user_count;
    RAISE NOTICE 'Rooms created: %', room_count;
    RAISE NOTICE 'Messages created: %', message_count;
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Demo Accounts (password: admin123):';
    RAISE NOTICE 'Admin: admin@windgo.com';
    RAISE NOTICE 'Demo User: demo@windgo.com';
    RAISE NOTICE '========================================';
    RAISE NOTICE 'Note: GORM will create tables automatically';
    RAISE NOTICE 'This script adds initial data after GORM migration';
    RAISE NOTICE '========================================';
END $$;
