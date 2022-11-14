# Process-Schedule-Go 流程调度引擎

Process-Schedule-Go(ps-go)是一个流程调度引擎,可以对流程进行可视化编排。

这听起来很不太容易理解，我们来想想这样一个需求场景，现在产品经理告诉你，针对某某节日，需要做一个针对于新用户的活动，希望能够在用户注册成功之后给用户发放一张优惠券，在发送短息通知用户。

一般在公司中，用户中心、优惠券中心、短信服务都是属于基础服务，都由分别的人进行负责。用户中心具有注册接口，优惠券中心具有优惠券发送接口，短信中心具有短信发送接口。

在这个需求背景下，可以由前端或者后端来进行解决，首先来说一下后端处理的方法，由用户中心去在用户注册接口去处理，当用户注册成功之后再去调用优惠券发放，短信通知接口，
但是用户中心本身就是一个独立的服务，再在里面加一些优惠券相关的代码，各个服务之间的耦合性太高，作为用户中心的服务负责人，他肯定也不愿意去为了这个需求而去改这个接口。
其次就是前端来进行解决，前端在调用用户注册接口成功之后，在调用优惠券发放和短信发放接口， 前端处理的坏处最直观的就是，你的分多次请求，请求次数越多，需要的时间越长，这里的请求都是从客户端直接请求，肯定没有从用户中心服务请求快，因为他们都是内网，而且客户端请求一次还要做判断。
当然前端也可以做异步请求，但是如果用户以为已经注册成功，从而关掉界面啥的，就会出问题。

针对于这个需求，调度引擎来进行处理就显得极为方便，我们来看看以下流程配置：
```
{
    "request": {
        "type": "json",
        "body": {
            "phone": {
                "type": "string",
                "required": true
            }
        }
    },
    "components": [
        [
            {
                "name": "user_register",
                "desc": "用户注册",
                "type": "api",
                "method":"get",
                "url": "http://user_service/api/v/1/register",
                "input": {
                    "phone": "{request.body.phone}"
                },
                "outputName": "userResponse",
                "timeout": 10
            },
            {
                "name": "grant_coupon",
                "desc": "发放优惠券",
                "type": "api",
                "method":"get",
                "url": "http://coupon_service/api/v/1/grant_coupon",
                "input": {
                    "user_id": "{user.id}"
                },
                "outputName": "grantCouponResponse",
                "timeout": 10
            },
            {
                "name": "send_msg",
                "desc": "发送短信",
                "type": "api",
                "method":"get",
                "url": "http://msg_service/api/v/1/send_msg",
                "input": {
                    "phone": "{request.body.phone}"
                },
                "outputName": "msgResponse",
                "timeout": 10
            }
        ]
    ],
    "response": {
        "type": "json",
        "body": {
            "code": "{response.code}",
            "msg": "{response.msg}",
            "data": "{response.data}"
        },
        "defaultBody": {
            "code": 200,
            "msg": "success",
            "data": "注册成功"
        }
    }
}
```

看到以上配置是不是还是一脸懵逼，没关系，接下来我们一步步讲解。

#### 流程主配置

流程主配置主要由请求配置（request）、组件配置（components）、返回配置（response）组成，主要结构如下：
```
{
    "record": true,     
    "suspend": false,   //是否支持任务挂起
    "request": {},      //请求相关配置
    "response": {},     //返回相关配置
    "components": [     //执行组件相关配置
        [
            {}
        ]
    ]
}

# 字段含义解释
record：是否记录执行日志，当流程执行完成之后，是否需要将执行日志写入数据库，可以用作后续的执行流程查询等。
suspend：是否支持任务挂起，任务挂起就是指当我们的执行流程比较长的情况下，若遇到代码bug\接口bug\网络波动等情况，导致流程异常时，是否支持恢复重试，此时流程会保存当前执行状态，写入数据库，可以等待异常解决之后进行手动恢复。
request：请求相关配置，后续详细说明。
response：返回相关配置，后续详细说明。
components：执行组件相关配置，后续详细说明。
```

#### 请求配置
在讲解请求配置之前，我们先来认识以下，字段验证配置(FieldValidate),字段验证配置时用来对字段进行限制的，具体配置如下：
```
{
    "type": "string",           //字段类型 [int|float|string|slice|bool|object]
    "attribute": FieldValidate, //子属性字段验证配置, 仅[object]支持
    "required": true,           //是否必填
    "default": "1",             //默认值
    "maxLen": 10,               //最大长度, 仅[string]支持
    "minLen": 2,                //最小长度, 仅[string]支持
    "max": 10,                  //最大值, 仅[int|float]支持
    "min": 4,                   //最小值, 仅[int|float]支持
    "enum": ["1","2"]           //枚举值, 仅[int|float|string|slice]支持
}
```
请求主配置字段：
```
{
    "type": "json",           //数据类型，比如发送post请求的时候，前端可能发送的时xml格式的数据，这时候type则填写xml
    "query": FieldValidate,   //通过url携带的参数规则校验
    "body": FieldValidate,    //通过body携带的参数规则校验
    "header": FieldValidate   //通过请求头携带的参数规则校验
}
```
我希望设置前端在请求的时候必须传一个数据为json，存在一个字段为phone，且必须为string类型的配置示例：
```
{
    "type": "json",
    "body": {
        "phone": {
            "type": "string",
            "required": true
        }
    }
}
```

#### 流程组件配置
流程组件配置是一个二维数组，是由多个组件配置组成的，格式为[][]component，示例如下：
```
[    
    [   // step-1
      { // action-1
        ...
      },
      { // action-2
        ...
      },
      ...//具体的component的配置规则
    ],
    
    [ // step-2
      { // action-1
        ...
      },
      { // action-2
        ...
      },
      ...//具体的component的配置规则
    ],
    
]

```
我们把这个而为数组的行称作step,把每一行中的配置叫做组件配置component。
在执行流程组件是，是从上往下执行的，第一层执行完成之后，执行第二层，第二层执行完成之后执行第三层...,知道流程结束。
而每一层中，具有N个组件配置，这些组件配置是真正需要执行的配置，每一层中的组件配置是通过并发进行执行的。当第一层的所有组件并发执行完成之后，才会执行第二层。

执行顺序从上往下，流程如下:
```
|-step-1->并发执行[action-1、action-2...]
|-step-2->并发执行[action-1、action-2...]   
|-step-3->并发执行[action-1、action-2...]
|-step-4->并发执行[action-1、action-2...]   
...

```

了解了执行规则之后，我们再来详细说一下组件配置，具体可配置字段如下：
```
{
    "name": "devops",  //组件名,同一个step层下，name不能重复
    "desc": "流程描述", //组件描述
    "type": "script", //组件类型 [api|script]
    "url": "rule/api/test2.js", //type=script时则为具体的脚本文件，否则为api的url
    "input": { //输入参数
        "data": "{request.body}" //{request.body}表示去输入的request配置下的body字段的值，也就是请求时携带的body数据
    },
    "condition":"{request.body.phone} != '0000'" //执行准入条件，当条件符合才进行执行脚本，否则跳过执行
    "isCache":true, //是否进行执行缓存，设置了之后，不会执行组件，直接取上一次的返回值。
    "outputName": "devops", //输出的对象名。假如你的接口或者脚本最终返回数据为{"code":200}。则后续可以通过{devops.code}进行取值
    "timeout": 10 //执行超时时间
    "retryMaxCount":1,//最大重试次数
    "retryMaxWait":10, //重试最大等待时长
	"method":"get",//请求方法，仅api支持
    "contentType":"", //数据类型，仅api支持
    "auth":["123","456"],//请求header auth，仅api支持
    "header":{},   //请求header头，仅api支持
    "responseType":"json/xml", //返回数据类型，仅api支持
    "dataType":"json/xml", //请求数据类型，仅api支持
    // {code:200,msg:"success",data:{phone:"xxxx"}}
    "responseCondition":"{code}==200", //返回条件判断
    "outputData":"{data}", //返回数据
    "errMsg":"{msg}",
    "tls":{       //发送http请求携带的证书
        "ca":"123",  //ca 标志符。会从密钥库查询标志符对应的密钥
        "key":"345"  //key 标志符。会从密钥库查询标志符对应的密钥
     }     
}
```

组件主要分为两种，一种是脚本组件，一种是api组件。脚本组件我们可以用它来进行复杂的判断等,我们也可以通过javascript来进行编写脚本。比如上面的配置执行了一个rule/api/test2.js的脚本。我们来看看这个脚本的代码
```
function handler(ctx,input){
    return ctx.request({"method":"get","url":"https://baidu.com"})
}
```
这是一个很简单的脚本代码，主要是发送了一个get请求。但是这里我们要注意的是，脚本文件具有固定的格式，也就是
```
function handler(ctx,input){
    // do some
    return xxx;
}
```
其中ctx可以用来调用一些系统提供的方法。比如刚才我们看到的ctx.request()就是发送请求的。
input是组件配置中input配置的输入的参数，函数return之后的数据就会被挂载到配置中的outputName设置的字段上。

ctx 目前提供的方法主要如下：
```
ctx.request //发送请求 ctx.request({})
ctx.log     //打印日志 ctx.log.info()
ctx.data    //设置上下文数据 ctx.data.load() ctx.data.store()
ctx.logId   //获取日志链路id ctx.logId()
ctx.trx     //获取唯一标志符 ctx.trx()
ctx.suspend //主动挂起流程 ctx.suspend("挂起code","挂起原因")
ctx.break   //主动中断流程 ctx.break("中断原因")
ctx.response //主动返回数据 ctx.response({})
ctx.base64  //base64加解密相关 ctx.base64.encode() ctx.base64.decode()
ctx.uuid    //生成唯一字符串 ctx.uuid()
ctx.aes     //aes加解密相关
ctx.rsa     //rsa加解密相关,关联密钥管理库

```
密钥管理库就是把所有的密钥信息进行统一管理，一个密钥存在一个对应的标志符。我们在对接一些接口的时候，存在需要使用密钥的情况，这种时候我们不需要在代码里面去处理密钥，直接使用密钥标志符就可以了，在代码里面会通过密钥标志符找到对应的密钥信息使用。


### 相关api
```
	// 流程规则相关api
		api.GET("/rule", handler.GetRule)   
		api.GET("/rule/page", handler.PageRule)
		api.POST("/rule", handler.AddRule)
		api.PUT("/rule/switch_version", handler.SwitchRuleVersion)
		api.DELETE("/rule", handler.DeleteRule)

		// 脚本相关api
		api.GET("/script", handler.GetScript)
		api.GET("/script/page", handler.PageScript)
		api.POST("/script", handler.AddScript)
		api.PUT("/script/switch_version", handler.SwitchScriptVersion)
		api.DELETE("/script", handler.DeleteScript)

		// 密钥管理相关
		api.GET("/secret", handler.GetSecret)
		api.GET("/secret/page", handler.PageSecret)
		api.POST("/script", handler.AddSecret)
		api.PUT("/secret", handler.UpdateSecret)
		api.DELETE("/script", handler.DeleteSecret)

		// 异常中断api
		api.GET("/suspend/page", handler.PageSuspend)
		api.GET("/suspend", handler.GetSuspend)
		api.POST("/suspend/recover", handler.SuspendRecover) //异常中断恢复
		api.PUT("/suspend", handler.UpdateSuspend)

		// 执行日志相关api
		api.GET("/run_log", handler.GetRunLog)
```
