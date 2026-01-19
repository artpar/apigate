# Database Setup

APIGate uses SQLite as its database with automatic schema migrations.

---

## SQLite Database

SQLite requires no setup and works for most deployments. Migrations run automatically on startup.

### Configuration

```bash
# Environment variable
APIGATE_DATABASE_DSN=apigate.db

# Or in config file
database:
  driver: sqlite
  dsn: "apigate.db"
```

The database file is created at the specified path. Default is `apigate.db` in the current directory.

### Built-in Optimizations

APIGate configures SQLite with the following optimizations automatically:

- **WAL mode** - Enables concurrent reads during writes
- **Busy timeout** - 5000ms wait on lock contention
- **Cache size** - 64MB for better performance
- **Synchronous mode** - NORMAL for safe performance balance
- **Temp store** - In-memory for faster temp tables

These are hardcoded for optimal performance; no configuration is required.

---

## Automatic Migrations

Schema migrations are embedded in the binary and run automatically when APIGate starts or during `apigate init`. There is no manual migration command.

### Migration Tracking

Migrations are tracked in the `schema_migrations` table:

```sql
SELECT * FROM schema_migrations;
-- Shows all applied migration versions
```

### Migration Files

Migration files are located in `adapters/sqlite/migrations/` in the source code:

```
001_initial.sql
002_add_password_hash.sql
003_routes.sql
004_auth_tokens.sql
005_settings.sql
...
020_certificates.sql
```

---

## Backup & Restore

### Backup

```bash
# Simple file copy (while APIGate is stopped)
cp apigate.db apigate-backup-$(date +%Y%m%d).db

# Or with SQLite backup command (can run while APIGate is running)
sqlite3 apigate.db ".backup 'backup.db'"
```

### Restore

```bash
# Stop APIGate first, then:
cp backup.db apigate.db
```

---

## Troubleshooting

### Database Locked

**Error**: `database is locked`

**Causes**:
- Another process has the database open
- Long-running transaction

**Solutions**:
- Ensure only one APIGate instance uses the database file
- Check for stale locks: `fuser apigate.db`
- The built-in 5-second busy timeout handles most temporary locks

### Database Corruption

**Error**: `database disk image is malformed`

**Solutions**:
1. Stop APIGate
2. Try recovery:
```bash
sqlite3 apigate.db ".recover" | sqlite3 recovered.db
```
3. If recovery works, replace the database file

### Permissions

**Error**: `unable to open database file`

**Solutions**:
- Ensure the database directory exists
- Check write permissions on the directory
- Verify the DSN path is correct

---

## See Also

- [[Configuration]] - Full configuration reference
