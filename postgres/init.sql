-- Dhaba Orders: database schema and seed data

CREATE TABLE IF NOT EXISTS menu_items (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(100)   NOT NULL UNIQUE,
    description TEXT,
    price       NUMERIC(10,2)  NOT NULL,
    category    VARCHAR(50),
    available   BOOLEAN        NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS orders (
    id            SERIAL PRIMARY KEY,
    customer_name VARCHAR(100)  NOT NULL,
    status        VARCHAR(20)   NOT NULL DEFAULT 'new'
                      CHECK (status IN ('new', 'cooking', 'ready')),
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id           SERIAL PRIMARY KEY,
    order_id     INTEGER        NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    menu_item_id INTEGER        NOT NULL REFERENCES menu_items(id),
    name         VARCHAR(100)   NOT NULL,
    price        NUMERIC(10,2)  NOT NULL,
    quantity     INTEGER        NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_orders_status     ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at ON orders(created_at);
CREATE INDEX IF NOT EXISTS idx_order_items_order ON order_items(order_id);

-- Keep updated_at current
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN NEW.updated_at = NOW(); RETURN NEW; END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_orders_updated_at ON orders;
CREATE TRIGGER trg_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Seed menu
INSERT INTO menu_items (name, description, price, category) VALUES
    ('Chicken Biryani',  'Fragrant basmati rice with slow-cooked spiced chicken',       450.00, 'Main Course'),
    ('Mutton Karahi',    'Tender mutton in a rich tomato and green-chilli gravy',        650.00, 'Main Course'),
    ('Dal Makhani',      'Black lentils simmered overnight with butter and cream',       250.00, 'Main Course'),
    ('Aloo Gobi',        'Potato and cauliflower stir-fried with cumin and turmeric',   200.00, 'Main Course'),
    ('Naan',             'Fresh leavened flatbread baked in the tandoor',                50.00, 'Bread'),
    ('Chapli Kebab',     'Peshawari spiced minced-meat patty, pan-fried',               180.00, 'Starter'),
    ('Seekh Kebab',      'Minced meat on skewers, grilled over charcoal',               200.00, 'Starter'),
    ('Paneer Tikka',     'Marinated cottage-cheese cubes grilled in the tandoor',       300.00, 'Starter'),
    ('Mango Lassi',      'Chilled yogurt drink blended with fresh Chaunsa mango',       120.00, 'Drinks'),
    ('Kheer',            'Rice pudding with cardamom, rose water and pistachios',       150.00, 'Dessert')
ON CONFLICT (name) DO NOTHING;
