package controller

import (
	"crack_front/src/common"
	"crack_front/src/master/logManager"
	"crack_front/src/master/middleware"
	"crack_front/src/master/taskManager"
	"crack_front/src/master/user"
	"crack_front/src/master/workerManager"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"strconv"
)

// Content-Type: application/json
// Authorization: token

// POST  用户注册
func Register(c *gin.Context) {
	var (
		newUser 		*common.User = &common.User{}
		realUser		*common.User
		matched			bool
		err 			error
		hashedPassword  []byte
		token			string
	)
	c.Bind(newUser)
	if newUser.UserId != 0 {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "UserId不可指定!",
		})
		return
	}
	fmt.Println(*newUser)
	// 校验用户名
	if  matched = user.VerifyUserName(newUser.UserName); !matched {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "用户名是8-16位由大小写英文字母、数字和下划线构成",
		})
		return
	}

	// 校验邮箱
	if  matched = user.VerifyUserEmail(newUser.UserEmail); !matched {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "邮箱格式非法",
		})
		return
	}

	// 校验密码
	if  matched = user.VerifyUserPassword(newUser.UserPassword); !matched {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "密码是8-16位由大小写英文字母、数字和下划线构成",
		})
		return
	}

	// 判断用户名是否存在
	if matched, err = user.NameExisted(newUser.UserName); err != nil || matched {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "用户名已注册!",
		})
		return
	}

	// 判断邮箱是否存在
	if matched, err = user.EmailExisted(newUser.UserEmail); err != nil || matched {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "邮箱已注册!",
		})
		return
	}

	if hashedPassword, err = bcrypt.GenerateFromPassword([]byte(newUser.UserPassword), bcrypt.DefaultCost); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "服务器内部出现错误,请稍候再试!",
		})
		return
	}

	newUser.UserPassword = string(hashedPassword)
	if err = user.InsertUser(newUser); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "服务器内部出现错误,请稍候再试!",
		})
		return
	}

	// 获取用户id
	if realUser, err = user.SelectUserByName(newUser.UserName); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 0,
			"message": "注册成功!发放token失败,请稍候再试!",
		})
	}

	if token, err = middleware.ReleaseToken(realUser); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "注册成功!发放token失败,请稍候再试!",
		})
	} else {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 0,
			"message": "注册成功,之后的通信需要携带token",
			"token": token,
		})
	}
}

// POST 用户登录
func Login(c *gin.Context) {
	var (
		curUser 		= &common.User{}
		realUser		*common.User
		err 			error
		token			string
		matched			bool
	)
	c.Bind(curUser)

	// 校验密码格式
	if  matched = user.VerifyUserPassword(curUser.UserPassword); !matched {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "密码是8-16位由大小写英文字母、数字和下划线构成",
		})
		return
	}

	if len(curUser.UserName) > 0 { // 校验用户名
		// 用户名登录
		if matched = user.VerifyUserName(curUser.UserName); !matched {
			c.JSON(http.StatusAccepted, gin.H{
				"errno":   1,
				"message": "用户名是8-16位由大小写英文字母、数字和下划线构成",
			})
			return
		}
		realUser, err = user.SelectUserByName(curUser.UserName)
	} else if len(curUser.UserEmail) > 0 { // 校验邮箱
		// 邮箱登录
		if matched = user.VerifyUserEmail(curUser.UserEmail); !matched {
			c.JSON(http.StatusAccepted, gin.H{
				"errno":   1,
				"message": "邮箱格式非法",
			})
			return
		}
		realUser, err = user.SelectUserByEmail(curUser.UserEmail)
	} else {
		c.JSON(http.StatusAccepted, gin.H{
			"errno":   1,
			"message": "请使用用户名或者邮箱登录!",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno":   1,
			"message": "用户不存在!",
		})
		return
	}

	// 密码是否正确
	if err = bcrypt.CompareHashAndPassword([]byte(realUser.UserPassword), []byte(curUser.UserPassword)); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno":   1,
			"message": "密码错误!",
		})
		return
	}

	// 发放token
	if token, err = middleware.ReleaseToken(realUser); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": "登录成功!发放token失败,请稍候再试!",
		})
	} else {
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 0,
			"message": "登录成功,之后的通信需要携带token",
			"token": token,
		})
	}
}


// POST 用户发起识别请求
func CrackIdentify(c *gin.Context)  {
	var(
		err     		error
		task    		= &common.Task{}
		userId			interface{}
		ok				bool
		finishChan		<-chan struct{}
		failChan		<-chan struct{}
		errChan			<-chan error
	)
	if err = c.BindJSON(task); err != nil{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message":err.Error(),
		})
		return
	}

	if !common.VerifyTaskType(task.TaskType) {
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message": "请使用正确的task_type['image', 'video']",
		})
		return
	}

	if !common.VerifyTaskName(task.TaskName) {
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message": "任务名称task_name不合法",
		})
		return
	}

	if userId, ok = c.Get("UserId"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"errno": 1,
			"message": "请先登录后携带token以获取UserId",
			"data":nil,
		})
		return
	}
	task.UserId = userId.(uint)

	// 插入任务
	if err = taskManager.TM.SaveTask(task); err != nil {
		c.JSON(http.StatusAccepted, gin.H{
			"errno":1,
			"message":err.Error(),
			"data": nil,
		})
	}

	// 等待任务完成
	finishChan, failChan, errChan = taskManager.TM.WatchTask(task)
	select {
	case err = <-errChan:
		// 通知错误
		c.JSON(http.StatusInternalServerError, gin.H{
			"errno":1,
			"message":err.Error(),
		})
	case <-failChan:
		// 通知识别出错
		c.JSON(http.StatusInternalServerError, gin.H{
			"errno":1,
			"message":err.Error(),
		})
	case <-finishChan:
		// 通知识别完成
		c.JSON(http.StatusOK, gin.H{
			"errno": 0,
			"message": "识别完成",
		})
	}
}


// DELETE 从etcd中删除任务
func RemoveTask(c *gin.Context)  {
	var (
		ok					bool
		err 				error
		task 				*common.Task
		userId				interface{}
	)

	if err = c.BindJSON(&task); err != nil{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message":err.Error(),
		})
		return
	}

	if userId, ok = c.Get("UserId"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"errno": 1,
			"message": "请先登录后携带token以获取UserId",
		})
		return
	}
	task.UserId = userId.(uint)

	if _, err = taskManager.TM.RemoveTask(task); err != nil{
		c.JSON(http.StatusAccepted, gin.H{
			"errno":1,
			"message":"success",
		})
	}else{
		c.JSON(http.StatusOK, gin.H{
			"errno":0,
			"message":"success",
		})
	}
}


// GET 获取etcd当前有哪些任务
func GetTasks(c *gin.Context)  {
	var (
		err 			error
		taskList 		[]*common.Task
	)

	if taskList, err = taskManager.TM.GetTaskList(); err != nil{
		c.JSON(http.StatusAccepted, gin.H{
			"errno":1,
			"message":err.Error(),
			"data":nil,
		})
	}else{
		c.JSON(http.StatusOK, gin.H{
			"errno":0,
			"message":"success",
			"data":taskList,
		})
	}
}

// POST 强制杀死任务进程
func KillTask(c *gin.Context)  {
	var (
		ok					bool
		err 				error
		task				*common.Task
		userId				interface{}
	)

	if err = c.BindJSON(&task); err != nil{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message":err.Error(),
		})
		return
	}

	if userId, ok = c.Get("UserId"); !ok {
		c.JSON(http.StatusUnauthorized, gin.H{
			"errno": 1,
			"message": "请先登录后携带token以获取UserId",
		})
		return
	}
	task.UserId = userId.(uint)

	if err = taskManager.TM.KillTask(task); err != nil{
		c.JSON(http.StatusAccepted, gin.H{
			"errno":1,
			"message":err.Error(),
		})
	}else{
		c.JSON(http.StatusOK, gin.H{
			"errno":0,
			"message":"success",
		})
	}
}

// GET 获取任务执行日志
func QueryTaskLog(c *gin.Context) {
	var(
		ok 				bool
		err 			error
		taskName		string
		skipStr			string
		skip 			int
		limitStr		string
		limit 			int
		taskLogList		[]*common.TaskLog
	)
	if taskName, ok = c.GetQuery("task_name"); !ok{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message":"缺少query字段:task_name",
			"data":nil,
		})
		return
	}
	if skipStr, ok = c.GetQuery("skip"); !ok{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message":"缺少query字段:skip",
			"data":nil,
		})
		return
	}
	if limitStr, ok = c.GetQuery("limit"); !ok{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message":"缺少query字段:limit",
			"data":nil,
		})
		return
	}

	if skip, err = strconv.Atoi(skipStr); err != nil{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message": err.Error(),
			"data":nil,
		})
		return
	}

	if limit, err = strconv.Atoi(limitStr); err != nil{
		c.JSON(http.StatusCreated, gin.H{
			"errno":1,
			"message":err.Error(),
			"data":nil,
		})
		return
	}

	if taskLogList, err = logManager.LM.QueryTaskLog(taskName, int64(skip), int64(limit)); err != nil{
		c.JSON(http.StatusAccepted, gin.H{
			"errno":1,
			"message":err.Error(),
			"data":nil,
		})
	}else{
		c.JSON(http.StatusOK, gin.H{
			"errno":0,
			"message":"success",
			"data":taskLogList,
		})
	}
}

// GET 获取健康的worker节点
func GetWorkers(c *gin.Context)  {
	var (
		err 			error
		workers			[]string
	)

	if workers, err = workerManager.WM.GetWorkers(); err != nil{
		c.JSON(http.StatusAccepted, gin.H{
			"errno": 1,
			"message": err.Error(),
			"data": nil,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"errno": 0,
			"message": "success",
			"data": workers,
		})
	}
}