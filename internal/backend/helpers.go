package backend

import (
	"database/sql"
	"fmt"
	"strings"
)

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

// insertObject adds new object to database and increment children counter (SIZE) of parent
func insertObject(data map[string]any) error {
	l := len(data)
	if l == 0 {
		return fmt.Errorf("no data to insert")
	}
	objectID := data["OBJECT_ID"].(string)
	parentID := data["PARENT_ID"].(string)

	cols := make([]string, l+1)
	marks := make([]string, l+1)
	args := make([]any, l+1)
	i := 0

	cols[i] = "LEVEL"
	marks[i] = "?"
	args[i] = strings.Count(objectID, "$")

	for k, v := range data {
		i++
		cols[i] = k
		marks[i] = "?"
		args[i] = v
	}

	q := "INSERT INTO" + " OBJECTS (" + strings.Join(cols, ",") + ") VALUES (" + strings.Join(marks, ",") + ")"
	if err := execQuery(nil, q, args...); err != nil {
		return err
	}

	if parentID != "-1" {
		q = `UPDATE OBJECTS SET SIZE = SIZE + 1 WHERE OBJECT_ID = ? AND TYPE = ?`
		return execQuery(nil, q, parentID, Folder)
	}

	return nil
}
