### 设置你的mc Github Repository
Fork [mc upstream](https://github.com/minio/mc/fork)源码仓库到你的个人仓库。
```
$ mkdir -p $GOPATH/src/github.com/minio
$ cd $GOPATH/src/github.com/minio
$ git clone https://github.com/$USER_ID/mc
$ cd mc
$ make
$ mc --help
```

###  开发者指南

``mc``欢迎各位有识之士的贡献。为了让大家配合的更默契，我们做出如下约定：

* 尽情的fork项目并修改，我们鼓励pull request来讨论代码的修改。
    - Fork项目
    - 创建你的特性分支(git checkout -b my-new-feature)
    - Commit你的修改(git commit -am 'Add some feature')
    - Push到你的远端 (git push origin my-new-feature)
    - 创建一个新的Pull Request

* 如果你有``mc``的更多依赖，``mc``使用[govendor](https://github.com/kardianos/govendor)管理它的依赖。
    - 运行`go get foo/bar`
    - 修改你的代码，引入foo/bar
    - 在根目录运行`make pkg-add PKG=foo/bar`

* 如果你需要从``mc``中删除依赖
    - 修改你的代码，不引入foo/bar
    - 在根目录运行`make pkg-remove PKG=foo/bar`

* 如果你准备提起一个pull request请确保：
    - 新写的代码有测试用例，如果你不知道咋弄，请在pull request中提出来。
    - 运行`go fmt`
    - 使用`git rebase -i`将你的多个commit合并成一个，你可以强制更新你的pull request。
    - 确保`make build`完成。

* 参考[Effective Go](https://github.com/golang/go/wiki/CodeReviewComments)
    - `mc`项目符合Golang风格。
    - 如果你发现有问题的代码，请随时提起issue或者pull request。
