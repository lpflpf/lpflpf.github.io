---
title: "类的sizeof"
date: 2023-08-25
type: cpp
tags:
  - "c++ 学习笔记"
  - "学习笔记"
---

## 一个空的类

空类的sizeof 为1B；

## 虚函数

**一个类中，如果有虚函数存在，则会新建一个表，存放这些虚函数。每个对象将在其首地址处创建一个虚函数指针，用来存放指向虚函数表的指针。**

如下图所示：
{{<figure src="类的虚函数.png" width="600">}}

### 验证工具
为了验证类的虚函数，首先定义一个打印和访问虚函数的方法:

```c++
typedef string (*Fun)();
Fun printPtr(void*, int, int);

// 通过指针直接调用虚函数表
Fun printPtr(void *ptr,int ptr_idx, int func_idx) {
  long * vptr = (long *)ptr; // 指向vptr的指针
  // 
  long *vtbl  = (long *)vptr[ptr_idx]; // 获取到指向 vtbl 的指针
  Fun f = (Fun)vtbl[func_idx];
  cout << (*f)() << " --> 0x" << hex << vtbl[func_idx] << " " << endl;
  return f;
}
```

写一段代码来做测试:
```c++
class A {
  public:
    virtual string func1(){ return "func1"; }
    virtual string func2(){ return "func2"; }
    A(int a):a(a) {}
  private:
    int a;
};

int main(){
  // sizeof A:16
  cout << "sizeof A: " << sizeof(A) << endl; 
  A* a = new A(10);

  // call virtual function
  // output: func1 --> 0x400f70
  printPtr(a, 0, 0);
  // output: func2 --> 0x400fd6
  printPtr(a, 0, 1);

  // 跳过首位的vptr, 直接取下一个位置的数据
  // output: 10
  cout <<dec << ((long*)((long*)a))[1] << endl;
}

```
从gdb的调试也可以看出端倪。
```
(gdb) p *a
$1 = {_vptr.A = 0x401130 <vtable for A+16>, a = 10}
(gdb) info vtbl a
vtable for 'A' @ 0x401130 (subobject @ 0x603010):
[0]: 0x400f70 <A::func1()>
[1]: 0x400fd6 <A::func2()>
```
a (0x603010) 是一个指针, 其对象位置(0x401130), 对象存了两个数据:
  - **_vptr.A** 即vptr，指向 vtable
  - a 对象中属性的值
vtable 中保存了两个虚函数的地址, func1(0x400f70)、func2(0x400fd6);

### 继承

### 单继承

**对于单继承，虚函数将都放置于子类的vPtr中，因此类对象的大小仍然是 vPtr的大小.**
{{<figure src="单继承.png" width="600">}}
代码如下：
```c++
class class_a {
	public:
		virtual string fun_a() { return "func a"; }
};

class class_b:virtual  public class_a {
	public:
		virtual string fun_b() { return "func b" ;}
};

class class_c :  virtual public class_b {
	public:
		virtual string fun_c() {return "func c";}
};

typedef string (*Fun)();
Fun printPtr(void*, int, int);

int main()
{
	class_a* a = new class_a;
	class_b* b = new class_b;
	class_c* c = new class_c;

	printPtr(c, 0, 0); // call fun_a
	printPtr(c, 0, 1); // call fun_b
	printPtr(c, 0, 2); // call fun_c
	return 0;
}
```

### 多重继承

**对于虚函数的多重继承, 子类会继承所有base类的vptr。类本身的virtual function会放在第一个类的虚表中**

如下图所示:

{{<figure src="多继承类的虚函数.png" width="600">}}

写一段代码来验证：

```c++
class class_a {
	public:
		virtual string fun_a() { return "func a"; }
};
class class_b {
	public:
		virtual string fun_b() { return "func b" ;}
};

class class_c :  public class_a,virtual public class_b {
	public:
		virtual string fun_c() {return "func c";}
};

int main()
{
	class_a* a = new class_a;
	class_b* b = new class_b;
	class_c* c = new class_c;

  // output: func a --> 0x400f6c
	printPtr(a, 0, 0);

  // output: func b --> 0x400fd2
	printPtr(b, 0, 0);

  // 输出和 class_a 的方法一致
  // output: func a --> 0x400f6c
	printPtr(c, 0,0);

  // 输出和 class_b 的方法一致
  // output: func b --> 0x400fd2 
	printPtr(c, 1,0);

  // output: func c --> 0x401038
	printPtr(c, 0,1);

	return 0;
}
```

用gdb 调试如下
```
(gdb) p	*c
$1 = {<class_a>	= {_vptr.class_a = 0x4011f8 <vtable for class_c+24>}, <class_b> = {
    _vptr.class_b = 0x401220 <vtable for class_c+64>}, <No data	fields>}
(gdb) info vtbl	c
vtable for 'class_c' @ 0x4011f8	(subobject @ 0x603050):
[0]: 0x400f6c <class_a::fun_a()>
[1]: 0x401038 <class_c::fun_c()>

vtable for 'class_b' @ 0x401220	(subobject @ 0x603058):
[0]: 0x400fd2 <class_b::fun_b()>
```

可以看出，class_c 继承了 class_a 、class_b 的vptr. 类c自定义的虚函数 func_c 被放在了第一个vptr的虚函数表中。因此，sizeof(class_c) 的结果是 16

### 多重继承的重载

#### 重载第一基类

**如果重载了第一个基类， vtable中的指针覆盖为子类的指针，其他无变化**
```c++
class class_a {
	public:
		virtual string fun_a() { return "func a"; }
};

class class_b{
	public:
		virtual string fun_b() { return "func b" ;}
};

class class_c :  virtual public class_b, virtual public class_a {
	public:
		virtual string fun_b() {return "func c_b";}
		virtual string fun_c() {return "func c";}
};

typedef string (*Fun)();
Fun printPtr(void*, int, int);

int main()
{
	class_a* a = new class_a;
	class_b* b = new class_b;
	class_c* c = new class_c;


  // c_b
	printPtr(c, 0, 0);
  // c
	printPtr(c, 0, 1);
  // a
	printPtr(c, 1, 0);

  // 16
	cout << dec << sizeof(class_c) << endl;
	return 0;
}
```

#### 重载非第一基类

代码例子:

```
class class_a {
	public:
		virtual string fun_a() { return "func a"; }
};

class class_b{
	public:
		virtual string fun_b() { return "func b" ;}
};

class class_c :  virtual public class_b, virtual public class_a {
	public:
		virtual string fun_a() {return "func c_a";}
		virtual string fun_c() {return "func c";}
};

int main()
{
	class_a* a = new class_a;
	class_b* b = new class_b;
	class_c* c = new class_c;

	printPtr(c, 0, 0); // func b
	printPtr(c, 0, 1); // c_a
 	printPtr(c, 0, 2); // func c

  printPtr(c, 1, 0); // segment fault

	cout << dec << sizeof(class_c) << endl; // 16

	return 0;
}
```
vtable 分布:
- 子类实例的分布
```
(gdb) info vtbl c
vtable for 'class_c' @ 0x401308 (subobject @ 0x603050):
[0]: 0x401046 <class_b::fun_b()>
[1]: 0x4010ac <class_c::fun_a()>
[2]: 0x40111a <class_c::fun_c()>

vtable for 'class_a' @ 0x401338 (subobject @ 0x603058):
[0]: 0x401111 <virtual thunk to class_c::fun_a()>
```
- 父类实例的分布
```
vtable for 'class_a' @ 0x401390 (subobject @ 0x603010):
[0]: 0x400fe0 <class_a::fun_a()>
```

从vtable 的分布可以看出，方法 func_a 挪到了 第一个vtable 中，但 出现了 `0x401111` 这个地址，既不是 class_a 的 func_a, 也不是class_c 的方法。
为什么会出线一个新的地址？([参考](https://blog.csdn.net/bobbypollo/article/details/79888455))

> **结论** 不在多重继承时，重载非第一位类的方法