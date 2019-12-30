/**
===========================================================================
 * Mysql数据库服务
===========================================================================
*/
package frame

import (
	"bytes"
	"database/sql"
	"model"
	_ "modules/mysql"
	"strconv"
	"strings"
	"time"
)

//* ================================ DEFINE ================================ */

type MysqlS struct {
	tag   string
	brain *BrainS

	Pool           model.SyncMapHub /* map[Token]*sql.DB */
	DefaultDBToken string
}

type DBConfS struct {
	Host     string
	User     string
	Pass     string
	Database string
}

//* ================================ PRIVATE ================================ */
func (mMysql *MysqlS) main() {
	// Initialize
	mMysql.Pool.Init("Mysql")
	// Default DB
	if mMysql.brain.Const.Database.Open {
		db := DBConfS{mMysql.brain.Const.Database.Host, mMysql.brain.Const.Database.User, mMysql.brain.Const.Database.Password, mMysql.brain.Const.Database.Database}
		mMysql.DefaultDBToken = mMysql.SetPool(db)
	}
}

//* 解析查询结果 */
func (mMysql *MysqlS) queryAnalyze(rows *sql.Rows) (int, interface{}) {
	/* 构造数据 */
	columns, err := rows.Columns()
	if err != nil {
		return 305, err
	}
	data := make([]interface{}, len(columns))
	pdata := make([]interface{}, len(columns))
	for i := range pdata {
		pdata[i] = &data[i]
	}
	// 录入数据
	finalData := model.SQLDataS{
		Column: columns,
	}
	finalData.Data = make([]interface{}, 0, 1024)
	for rows.Next() {
		err := rows.Scan(pdata...)
		if err != nil {
			return 305, err
		}
		dataA := make([]interface{}, 0, len(columns))
		dataA = append(dataA, data...)
		for k, v := range dataA {
			if v == nil {
				dataA[k] = ""
			} else {
				dataA[k] = string(v.([]byte))
			}
		}
		finalData.Data = append(finalData.Data, dataA)
	}
	return 100, finalData
}

//* 生成数据库Token */
func (mMysql *MysqlS) generateToken(db DBConfS) string {
	var buf bytes.Buffer
	buf.WriteString(db.Host)
	buf.WriteString(db.User)
	buf.WriteString(db.Pass)
	buf.WriteString(db.Database)
	buf.WriteString(mMysql.brain.Const.SystemSplit)
	buf.Write(mMysql.brain.SystemSalt())
	return mMysql.brain.Sha1Encode(buf.Bytes())
}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mMysql *MysqlS) Ontology(neuron *NeuronS) *MysqlS {
	mMysql.tag = "MysqlDriver"
	mMysql.brain = neuron.Brain
	mMysql.brain.SafeFunction(mMysql.main)
	return mMysql
}

//* 新建连接池 */
func (mMysql *MysqlS) SetPool(dbConf DBConfS) string {
	db, err := sql.Open("mysql", dbConf.User+":"+dbConf.Pass+"@tcp("+dbConf.Host+")/"+dbConf.Database+"?charset=utf8&multiStatements=true")
	if err != nil {
		mMysql.brain.MessageHandler(mMysql.tag, "SetPool", 300, err)
		return ""
	}
	mMysql.brain.MessageHandler(mMysql.tag, "SetPool", 100, "Default SQL Pool -> Connected")
	db.SetMaxIdleConns(30)
	db.SetConnMaxLifetime(300 * time.Second)
	token := mMysql.generateToken(dbConf)
	mMysql.Pool.Set(token, db)
	return token
}

//* 获取连接池 */
func (mMysql *MysqlS) GetPool(token string) *sql.DB {
	dbI := mMysql.Pool.Get(token)
	db, found := dbI.(*sql.DB)
	if !found {
		return nil
	}
	return db
}

//* 执行查询 */
func (mMysql *MysqlS) ExecQuery(sqlStr string, callback func(code int, data interface{}), tokens ...string) {
	/* 参数检验 */
	if mMysql == nil {
		callback(300, "ExecQuery -> Pool is Null")
		return
	}
	if mMysql.brain.CheckIsNull(sqlStr) {
		callback(200, "ExecQuery -> SQL String is Null")
		return
	}
	if mMysql.brain.Const.Database.Log {
		mMysql.brain.LogGenerater(model.LogWarn, mMysql.tag, "ExecQuery", sqlStr)
	}
	token := mMysql.DefaultDBToken
	if len(tokens) > 0 {
		token = tokens[0]
	}
	db, found := mMysql.Pool.Get(token).(*sql.DB)
	if !found {
		callback(300, "ExecQuery -> Pool is Null")
		return
	}
	codeC := make(chan int)
	dataC := make(chan interface{})
	defer close(codeC)
	defer close(dataC)
	go mMysql.brain.SafeFunction(func() {
		rows, err := db.Query(sqlStr)
		if err != nil {
			codeC <- 301
			dataC <- err
			return
		}
		code, data := mMysql.queryAnalyze(rows)
		rows.Close()
		codeC <- code
		dataC <- data
	})
	callback(<-codeC, <-dataC)
}

//* 执行事务(可查询) */
/*
 *
 * @param sql_task => 任务信息[Object]
 * example:{
 * task_a : 'select count(*) from table_a',
 * task_b : 'select count(*) from table_b',
 * task_c : 'select count(*) from table_c',
 * }
 * @param callback => 结果回调[code, result]
 */
func (mMysql *MysqlS) ExecTrans(sqlArray []string, callback func(code int, data interface{}), tokens ...string) {
	/* 参数检验 */
	if mMysql == nil {
		callback(300, "ExecTrans -> Pool is Null")
		return
	}
	if mMysql.brain.CheckIsNull(sqlArray) {
		callback(200, "ExecTrans -> SQL is Null")
		return
	}
	token := mMysql.DefaultDBToken
	if len(tokens) > 0 {
		token = tokens[0]
	}
	db, found := mMysql.Pool.Get(token).(*sql.DB)
	if !found {
		callback(300, "ExecQuery -> Pool is Null")
		return
	}
	codeC := make(chan int)
	dataC := make(chan interface{})
	defer close(codeC)
	defer close(dataC)
	go mMysql.brain.SafeFunction(func() {
		tx, err := db.Begin()
		if err != nil {
			codeC <- 302
			dataC <- err
			return
		}
		resPool := make(map[string]interface{}, len(sqlArray))
		for k, v := range sqlArray {
			if mMysql.brain.Const.Database.Log {
				mMysql.brain.LogGenerater(model.LogWarn, mMysql.tag, "ExecTrans", v)
			}
			if mMysql.brain.Const.RunEnv < 2 {
				mMysql.brain.LogGenerater(model.LogDebug, mMysql.tag, "ExecTrans", "[Running TransId] -> "+strconv.Itoa(k))
			}
			if mMysql.brain.CheckIsNull(v) {
				continue
			}
			if strings.ToUpper(v[:1]) == "S" {
				rows, err := tx.Query(v)
				if err != nil {
					rows.Close()
					resPool[sqlArray[k]] = err
					errRollback := tx.Rollback()
					if errRollback != nil {
						resPool[sqlArray[k]] = errRollback
						codeC <- 303
						dataC <- errRollback
						return
					}
					codeC <- 301
					dataC <- err
					return
				} else {
					_, data := mMysql.queryAnalyze(rows)
					rows.Close()
					resPool[sqlArray[k]] = data
				}
			} else {
				result, err := tx.Exec(v)
				if err != nil {
					resPool[sqlArray[k]] = err
					errRollback := tx.Rollback()
					if errRollback != nil {
						resPool[sqlArray[k]] = errRollback
						codeC <- 303
						dataC <- errRollback
						return
					}
					codeC <- 302
					dataC <- err
					return
				} else {
					rowsAffect, err := result.RowsAffected()
					if err != nil {
						resPool[sqlArray[k]] = err
					}
					resPool[sqlArray[k]] = rowsAffect
				}
			}
			if mMysql.brain.Const.RunEnv < 2 {
				mMysql.brain.LogGenerater(model.LogDebug, mMysql.tag, "ExecTrans", "[Finished TransId] -> "+strconv.Itoa(k))
			}
		}
		errCommit := tx.Commit()
		if errCommit != nil {
			codeC <- 304
			dataC <- errCommit
			return
		}
		codeC <- 100
		dataC <- resPool
	})
	callback(<-codeC, <-dataC)
}
