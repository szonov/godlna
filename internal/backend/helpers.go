package backend

import "database/sql"

// execQuery executes query only if err is nil and returns err
func execQuery(err error, query string, args ...interface{}) error {
	if err != nil {
		return err
	}
	_, err = DB.Exec(query, args...)
	return err
}

// execQueryRowsAffected executes query only if err is nil and returns amount of affected rows and err
func execQueryRowsAffected(err error, query string, args ...interface{}) (int64, error) {
	if err != nil {
		return 0, err
	}
	var res sql.Result
	res, err = DB.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
