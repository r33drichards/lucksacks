-- Drop existing tables to start fresh
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS product_categories;
DROP TABLE IF EXISTS categories;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS addresses;
DROP TABLE IF EXISTS users;


-- users table
CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- addresses table
CREATE TABLE addresses (
    address_id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
    address_line1 VARCHAR(255) NOT NULL,
    address_line2 VARCHAR(255),
    city VARCHAR(100) NOT NULL,
    state VARCHAR(100) NOT NULL,
    postal_code VARCHAR(20) NOT NULL,
    country VARCHAR(50) NOT NULL,
    is_default BOOLEAN DEFAULT false
);

-- products table
CREATE TABLE products (
    product_id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    price NUMERIC(10, 2) NOT NULL,
    stock INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- categories table
CREATE TABLE categories (
    category_id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL
);

-- product_categories linking table
CREATE TABLE product_categories (
    product_id INTEGER REFERENCES products(product_id) ON DELETE CASCADE,
    category_id INTEGER REFERENCES categories(category_id) ON DELETE CASCADE,
    PRIMARY KEY (product_id, category_id)
);

-- reviews table
CREATE TABLE reviews (
    review_id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES products(product_id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- orders table
CREATE TABLE orders (
    order_id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(user_id) ON DELETE CASCADE,
    order_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(50) NOT NULL,
    total_amount NUMERIC(10, 2) NOT NULL
);

-- order_items table
CREATE TABLE order_items (
    order_item_id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(order_id) ON DELETE CASCADE,
    product_id INTEGER REFERENCES products(product_id),
    quantity INTEGER NOT NULL,
    price NUMERIC(10, 2) NOT NULL
);

-- payments table
CREATE TABLE payments (
    payment_id SERIAL PRIMARY KEY,
    order_id INTEGER REFERENCES orders(order_id) ON DELETE CASCADE,
    payment_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    amount NUMERIC(10, 2) NOT NULL,
    payment_method VARCHAR(50) NOT NULL
);

-- Insert data

-- Users
INSERT INTO users (username, email, password) VALUES
('johndoe', 'johndoe@example.com', 'password123'),
('janedoe', 'janedoe@example.com', 'password123'),
('alicep', 'alicep@example.com', 'password123'),
('bobs', 'bobs@example.com', 'password123'),
('charliec', 'charliec@example.com', 'password123');

-- Addresses
INSERT INTO addresses (user_id, address_line1, city, state, postal_code, country, is_default) VALUES
(1, '123 Maple Street', 'Maplewood', 'New Jersey', '07040', 'USA', true),
(2, '456 Oak Avenue', 'Oakville', 'California', '94562', 'USA', true),
(3, '789 Pine Lane', 'Pinewood', 'Texas', '75001', 'USA', true),
(4, '101 Elm Court', 'Elmwood', 'Florida', '32789', 'USA', true),
(5, '212 Birch Road', 'Birchwood', 'Washington', '98001', 'USA', true);

-- Categories
INSERT INTO categories (name) VALUES
('Electronics'),
('Books'),
('Clothing'),
('Home Goods'),
('Sports');

-- Products
INSERT INTO products (name, description, price, stock) VALUES
('Laptop', 'High-performance laptop', 1500.00, 50),
('Smartphone', 'Latest generation smartphone', 999.99, 150),
('The Great Gatsby', 'A classic novel by F. Scott Fitzgerald', 15.50, 200),
('T-Shirt', '100% cotton t-shirt', 25.00, 500),
('Coffee Maker', 'Drip coffee maker with timer', 75.25, 80),
('Basketball', 'Official size and weight basketball', 29.99, 120),
('Desk Chair', 'Ergonomic office chair', 250.00, 30);

-- Product Categories
INSERT INTO product_categories (product_id, category_id) VALUES
(1, 1),
(2, 1),
(3, 2),
(4, 3),
(5, 4),
(6, 5),
(7, 4);

-- Orders and Order Items
-- Order 1 for John Doe
INSERT INTO orders (user_id, status, total_amount) VALUES (1, 'Completed', 1525.00);
INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
(1, 1, 1, 1500.00),
(1, 4, 1, 25.00);
INSERT INTO payments (order_id, amount, payment_method) VALUES (1, 1525.00, 'Credit Card');

-- Order 2 for Jane Doe
INSERT INTO orders (user_id, status, total_amount) VALUES (2, 'Shipped', 1015.49);
INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
(2, 2, 1, 999.99),
(2, 3, 1, 15.50);
INSERT INTO payments (order_id, amount, payment_method) VALUES (2, 1015.49, 'PayPal');

-- Order 3 for Alice
INSERT INTO orders (user_id, status, total_amount) VALUES (3, 'Processing', 105.24);
INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
(3, 5, 1, 75.25),
(3, 6, 1, 29.99);
INSERT INTO payments (order_id, amount, payment_method) VALUES (3, 105.24, 'Credit Card');

-- Order 4 for Bob
INSERT INTO orders (user_id, status, total_amount) VALUES (4, 'Cancelled', 275.00);
INSERT INTO order_items (order_id, product_id, quantity, price) VALUES
(4, 7, 1, 250.00),
(4, 4, 1, 25.00);
-- No payment for cancelled order

-- Reviews
INSERT INTO reviews (product_id, user_id, rating, comment) VALUES
(1, 1, 5, 'Absolutely love this laptop! Worth every penny.'),
(2, 2, 4, 'Great phone, but battery life could be better.'),
(3, 2, 5, 'A timeless classic. Highly recommend.'),
(4, 1, 3, 'The t-shirt is okay, but it shrunk after one wash.'),
(5, 3, 4, 'Makes great coffee, easy to use.'),
(6, 4, 5, 'Perfect basketball for outdoor courts.'),
(7, 5, 2, 'Chair is not as comfortable as I expected.'); 