/**
===========================================================================
 * Redis数据库服务
===========================================================================
*/
package frame

import (
	"modules/redigo/redis"
	"strconv"
	"time"
)

//* ================================ DEFINE ================================ */

type RedisS struct {
	tag   string
	brain *BrainS
	Pool  *redis.Pool
}

//* ================================ PRIVATE ================================ */
func (mRedis *RedisS) main() {
	mRedis.Pool = mRedis.newPool()
}

func (mRedis *RedisS) newPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     30,
		IdleTimeout: 300 * time.Second,
		Dial: func() (redis.Conn, error) {
			addr := mRedis.brain.Const.Redis.Host + ":" + strconv.Itoa(mRedis.brain.Const.Redis.Port)
			c, err := redis.Dial("tcp", addr)
			if err != nil {
				mRedis.brain.MessageHandler(mRedis.tag, "Dial", 400, err)
				return nil, err
			} else {
				mRedis.brain.MessageHandler(mRedis.tag, "Dial", 100, "Redis Connected")
			}
			if _, err := c.Do("AUTH", mRedis.brain.Const.Redis.Password); err != nil {
				mRedis.brain.MessageHandler(mRedis.tag, "Auth", 401, err)
				c.Close()
				return nil, err
			} else {
				mRedis.brain.MessageHandler(mRedis.tag, "Auth", 100, "Redis Auth Passed")
			}
			return c, err
		},
	}
}

//* ================================ PUBLIC ================================ */

//* 构造本体 */
func (mRedis *RedisS) Ontology(neuron *NeuronS) *RedisS {
	mRedis.tag = "RedisDriver"
	mRedis.brain = neuron.Brain
	mRedis.brain.SafeFunction(mRedis.main)
	return mRedis
}
