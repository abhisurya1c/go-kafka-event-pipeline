-- Create the database if it doesn't exist
IF NOT EXISTS (SELECT name FROM sys.databases WHERE name = 'EventsDB')
BEGIN
    CREATE DATABASE EventsDB;
END
GO

USE EventsDB;
GO

-- Users table
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[Users]') AND type in (N'U'))
BEGIN
    CREATE TABLE Users (
        Id NVARCHAR(50) PRIMARY KEY,
        Name NVARCHAR(MAX),
        Email NVARCHAR(MAX),
        CreatedAt DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET()
    );
END
GO

-- Orders table
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[Orders]') AND type in (N'U'))
BEGIN
    CREATE TABLE Orders (
        Id NVARCHAR(50) PRIMARY KEY,
        UserId NVARCHAR(50),
        Status NVARCHAR(50),
        Amount DECIMAL(18, 2),
        CreatedAt DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET(),
        CONSTRAINT FK_Orders_Users FOREIGN KEY (UserId) REFERENCES Users(Id)
    );
END
GO

-- Payments table
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[Payments]') AND type in (N'U'))
BEGIN
    CREATE TABLE Payments (
        OrderId NVARCHAR(50) PRIMARY KEY,
        Status NVARCHAR(50),
        Amount DECIMAL(18, 2),
        SettledAt DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET(),
        CONSTRAINT FK_Payments_Orders FOREIGN KEY (OrderId) REFERENCES Orders(Id)
    );
END
GO

-- Inventory table
IF NOT EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'[dbo].[Inventory]') AND type in (N'U'))
BEGIN
    CREATE TABLE Inventory (
        SKU NVARCHAR(50) PRIMARY KEY,
        Quantity INT,
        LastUpdated DATETIMEOFFSET DEFAULT SYSDATETIMEOFFSET()
    );
END
GO
