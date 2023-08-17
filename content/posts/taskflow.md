---
title: Taskflow 概览
subtitle:
date: 2023-08-10
author:
  name: huber
description: 针对 taskflow 的调研
keywords: c++ 工作流的调研
summary: 针对多核任务工作流的调研
---

## 主要功能

  - 实现了一种基于任务拓扑图的调度算法
  - 支持对cpu + gpu的混合调度
  - 拓扑图支持条件语句、循环语句、switch case 等简易逻辑
  - 支持模块级别的子流程
  - 支持动态任务（任务执行运行时添加任务）
  - 支持设置任务的优先级
  - 支持任务的profile查看

## 缺点

  - 数据传递没有实现传递途径，需要自行实现

## 如何配置化

## 实现方式

在启动执行器时，会直接启动固定大小的线程，用于做任务执行。在执行器上增加工作流后，开始进行计算。

主要有几个**名词**：

  - executor 执行器，一个executor 上会有多个worker线程
  - worker 工作线程
  - topology 拓扑图，拓扑图上会有一个或者多个node节点
  - task 是提交的任务
  - taskflow 实现的目标是，将task 在worker上并行执行，提升执行效率。

{{<figure src="golang-db-scanner-valuer-interface.png" title="taskflow 的任务调度图"   >}}

如调度图所示，每个worker 占用一个real core，每个core 会有两个 task 队列（gpu & cpu）。[core 只和自己的task 队列交互。](https://github.com/taskflow/taskflow/blob/9316d98937e992968f1fb3a5836bf3500f756df7/taskflow/core/executor.hpp#L1416)

steal 的task 应该是 wsq 的最外面的task

## 如何使用

[cookbook](https://taskflow.github.io/taskflow/Cookbook.html)

### 静态任务

[case](https://taskflow.github.io/taskflow/StaticTasking.html)
#### 代码示意

```c++
#include <taskflow/taskflow.hpp>  // Taskflow is header-only

int main(){
  
  // 启动执行器
  tf::Executor executor;

  // 定义任务流
  tf::Taskflow taskflow;
  auto [A, B, C, D] = taskflow.emplace(  // create four tasks
    [] () { std::cout << "TaskA\n"; },
    [] () { std::cout << "TaskB\n"; },
    [] () { std::cout << "TaskC\n"; },
    [] () { std::cout << "TaskD\n"; } 
  );                                  

  A.precede(B, C);  // A runs before B and C
  D.succeed(B, C);  // D runs after  B and C
                                      
  // 任务执行器启动，并执行任务流
  executor.run(taskflow).wait(); 

  return 0;
}
```

#### 工作流
{{< mermaid >}}
graph LR;

A --> B --> D
A --> C --> D
{{< /mermaid >}}

### Executor
[case](https://taskflow.github.io/taskflow/ExecuteTaskflow.html)
### 动态任务
[case](https://taskflow.github.io/taskflow/DynamicTasking.html)
> 动态任务支持在运行时，增加子任务
#### 代码示意
```c++
 tf::Taskflow taskflow;
 tf::Executor executor;

 tf::Task A = taskflow.emplace([] () {}).name("A");  // static task A
 tf::Task C = taskflow.emplace([] () {}).name("C");  // static task C
 tf::Task D = taskflow.emplace([] () {}).name("D");  // static task D

 tf::Task B = taskflow.emplace([] (tf::Subflow& subflow) { 
   tf::Task B1 = subflow.emplace([] () {}).name("B1");  // dynamic task B1
   tf::Task B2 = subflow.emplace([] () {}).name("B2");  // dynamic task B2
   tf::Task B3 = subflow.emplace([] () {}).name("B3");  // dynamic task B3
   B1.precede(B3);  // B1 runs bofore B3
   B2.precede(B3);  // B2 runs before B3
 }).name("B");

 A.precede(B);  // B runs after A
 A.precede(C);  // C runs after A
 B.precede(D);  // D runs after B
 C.precede(D);  // D runs after C

 executor.run(taskflow).get();  // execute the graph to spawn the subflow
 taskflow.dump(std::cout);      // dump the taskflow to a DOT format
```
#### 工作流
{{<mermaid>}}
graph LR;

subgraph SubFlow:B
B1 --> B3
B2 --> B3
B3 --> B
end

A --> C --> D
A --> B --> D

{{</mermaid>}}


### 条件任务
[case](https://taskflow.github.io/taskflow/ConditionalTasking.html)

> 支持多种条件的判断，因此可以支持if/while等的语意描述。

#### 代码示意

```c++
tf::Executor executor;
tf::Taskflow taskflow;

auto A = taskflow.emplace([&]() -> tf::SmallVector<int> { 
  std::cout << "A\n"; 
  return {0, 2};
}).name("A");
auto B = taskflow.emplace([&](){ std::cout << "B\n"; }).name("B");
auto C = taskflow.emplace([&](){ std::cout << "C\n"; }).name("C");
auto D = taskflow.emplace([&](){ std::cout << "D\n"; }).name("D");

A.precede(B, C, D);

executor.run(taskflow).wait();
```

#### 流程图

{{<mermaid>}}
graph TB;

A --> |0| B
A --> |1| C
A --> |2| D
{{</mermaid>}}

### 任务组合
[case](https://taskflow.github.io/taskflow/ComposableTasking.html)

```c++
// f1 has three independent tasks
tf::Taskflow f1;
f1.name("F1");
tf::Task f1A = f1.emplace([&](){ std::cout << "F1 TaskA\n"; });
tf::Task f1B = f1.emplace([&](){ std::cout << "F1 TaskB\n"; });
tf::Task f1C = f1.emplace([&](){ std::cout << "F1 TaskC\n"; });

f1A.name("f1A");
f1B.name("f1B");
f1C.name("f1C");
f1A.precede(f1C);
f1B.precede(f1C);

// f2A ---
//        |----> f2C ----> f1_module_task ----> f2D
// f2B --- 
tf::Taskflow f2;
f2.name("F2");
tf::Task f2A = f2.emplace([&](){ std::cout << "  F2 TaskA\n"; });
tf::Task f2B = f2.emplace([&](){ std::cout << "  F2 TaskB\n"; });
tf::Task f2C = f2.emplace([&](){ std::cout << "  F2 TaskC\n"; });
tf::Task f2D = f2.emplace([&](){ std::cout << "  F2 TaskD\n"; });

f2A.name("f2A");
f2B.name("f2B");
f2C.name("f2C");
f2D.name("f2D");

f2A.precede(f2C);
f2B.precede(f2C);

tf::Task f1_module_task = f2.composed_of(f1).name("module");
f2C.precede(f1_module_task);
f1_module_task.precede(f2D);

f2.dump(std::cout);
```

#### 流程图

{{<mermaid>}}
graph TB;

subgraph Taskflow:F2
f2B --> f2C
f2A --> f2C
f2C --> Taskflow:F1 --> f2D
end
subgraph Taskflow:F1
f1B --> f1C
f1A --> f1C
end
{{</mermaid>}}


{{< admonition type=danger title="警告" open=true >}}
不能并行执行同一个任务组合
{{</admonition>}}

### 异步任务
- 支持在subflow,executor,runtime 等级别的等待`join`
- 支持有依赖的异步任务，并支持在多线程中创建异步任务
- [case](https://taskflow.github.io/taskflow/DynamicTasking) | [有依赖的异步任务](https://taskflow.github.io/taskflow/DependentAsyncTasking)

#### 简单的异步任务
```c++
std::future<int> future = executor.async([](){ return 1; });
assert(future.get() == 1);
```

#### 多线程中的任务依赖
```c++
tf::Executor executor;

// main thread creates a dependent async task A
tf::AsyncTask A = executor.silent_dependent_async([](){});

// spawn a new thread to create an async task B that runs after A
std::thread t1([&](){
  tf::AsyncTask B = executor.silent_dependent_async([](){}, A);
});

// spawn a new thread to create an async task C that runs after A
std::thread t2([&](){
  tf::AsyncTask C = executor.silent_dependent_async([](){}, A);
});

executor.wait_for_all();
t1.join();
t2.join();
```
### 与runtime的交互
[case](https://taskflow.github.io/taskflow/RuntimeTasking)
支持task传入runtime参数，用runtime 参数调度手动执行调度
```c++
tf::Task A, B, C, D;
std::tie(A, B, C, D) = taskflow.emplace(
  [] () { return 0; },
  [&C] (tf::Runtime& rt) {  // C must be captured by reference
    std::cout << "B\n"; 
    rt.schedule(C); // b 唤起C
  },
  [] () { std::cout << "C\n"; },
  [] () { std::cout << "D\n"; }
);
A.precede(B, C, D);
executor.run(taskflow).wait();
```
###  优先级任务
[case](https://taskflow.github.io/taskflow/PrioritizedTasking)

```c++
tf::Executor executor(1);
tf::Taskflow taskflow;

int counter = 0;

auto [A, B, C, D, E] = taskflow.emplace(
  [] () { },
  [&] () { 
    std::cout << "Task B: " << counter++ << '\n';  // 0
  },
  [&] () { 
    std::cout << "Task C: " << counter++ << '\n';  // 2
  },
  [&] () { 
    std::cout << "Task D: " << counter++ << '\n';  // 1
  },
  [] () { }
);

A.precede(B, C, D); 
E.succeed(B, C, D);

B.priority(tf::TaskPriority::HIGH);
C.priority(tf::TaskPriority::LOW);
D.priority(tf::TaskPriority::NORMAL);

executor.run(taskflow).wait();
```
### gpu 任务
[case](https://taskflow.github.io/taskflow/GPUTaskingcudaFlow)
### 设置最大并发
[case](https://taskflow.github.io/taskflow/LimitTheMaximumConcurrency)
### 取消请求
[case](https://taskflow.github.io/taskflow/RequestCancellation)
### Profile
[case](https://taskflow.github.io/taskflow/ProfileProfilerr.html)

## 代码学习

### node 描述

```c++

// 定义不同的执行类型
struct Static {
  template <typename C>
  Static(C&&);
  std::variant<
    std::function<void()>, std::function<void(Runtime&)>
  > work;
};

struct Dynamic {
  template <typename C>
  Dynamic(C&&);

  std::function<void(Subflow&)> work;
  Graph subgraph;
};
// ...


using handle_t = std::variant<
  Placeholder,      // placeholder
  Static,           // static tasking
  Dynamic,          // dynamic tasking
  Condition,        // conditional tasking
  MultiCondition,   // multi-conditional tasking
  Module,           // composable tasking
  Async,            // async tasking
  DependentAsync    // dependent async tasking (no future)
>;


class Node {
  SmallVector<Node*> _successors; // node 执行成功后，需要转移的node 列表（map)
  SmallVector<Node*> _dependents; // node 依赖的其他node   
  handle_t _handle;
  unsigned _priority {0};
  Topology* _topology {nullptr};
  Node* _parent {nullptr};
}
```

### 初始化执行器
```c++
// N 是需要启动的执行线程的数量
inline Executor::Executor(size_t N, std::shared_ptr<WorkerInterface> wix) :
    _MAX_STEALS {((N+1) << 1)},
    _threads    {N},
    _workers    {N},
    _notifier   {N},
    _worker_interface {std::move(wix)} {
 
    if(N == 0) {
      TF_THROW("no cpu workers to execute taskflows");
    }
 
    _spawn(N);
 
    // instantite the default observer if requested
    if(has_env(TF_ENABLE_PROFILER)) {
      TFProfManager::get()._manage(make_observer<TFProfObserver>());
    }
  }
```

### 启动线程

```c++
inline void Executor::_spawn(size_t N) {
 
  std::mutex mutex;
  std::condition_variable cond;
  size_t n=0;
  // workers 保存的就是线程，创建N个
  for(size_t id=0; id<N; ++id) {
 
    _workers[id]._id = id;
    _workers[id]._vtm = id;
    _workers[id]._executor = this;
    _workers[id]._waiter = &_notifier._waiters[id];
     
    _threads[id] = std::thread([this] (
      Worker& w, std::mutex& mutex, std::condition_variable& cond, size_t& n
    ) -> void {
 
      w._thread = &_threads[w._id];
 
      {
        std::scoped_lock lock(mutex);
        _wids[std::this_thread::get_id()] = w._id;
        if(n++; n == num_workers()) {
          // 确保至少有一个worker在启动中
          cond.notify_one();
        }
      }
      Node* t = nullptr;
 
      // worker 启动前的钩子
      if(_worker_interface) {
        _worker_interface->scheduler_prologue(w);
      }
 
      std::exception_ptr ptr{nullptr};    
      try {
        while(1) {
          循环获取task和执行task
          // execute the tasks.
          _exploit_task(w, t);
 
          // wait for tasks
          if(_wait_for_task(w, t) == false) {
            break;
          }
        }
      }
      catch(...) {
        ptr = std::current_exception();
      }
 
      // worker 结束前的钩子
      if(_worker_interface) {
        _worker_interface->scheduler_epilogue(w, ptr);
      }
 
    }, std::ref(_workers[id]), std::ref(mutex), std::ref(cond), std::ref(n));
  }
 
  std::unique_lock<std::mutex> lock(mutex);
  cond.wait(lock, [&](){ return n==N; });
}
```

### 任务执行

```c++
// 循环获取task
  inline void Executor::_exploit_task(Worker& w, Node*& t) {
    while(t) {
      _invoke(w, t);
      // 每个worker 中有一个 _wsq 保存task列表
      t = w._wsq.pop();
    }
  }
 
 
// 处理task的过程
inline void Executor::_invoke(Worker& worker, Node* node) {
 
  // synchronize all outstanding memory operations caused by reordering
  while(!(node->_state.load(std::memory_order_acquire) & Node::READY));
 
  begin_invoke:
   
  SmallVector<int> conds;
 
  // 取消直接返回
  if(node->_is_cancelled()) {
    if(node = _tear_down_invoke(worker, node); node) {
      goto invoke_successors;
    }
    return;
  }
 
  // if acquiring semaphore(s) exists, acquire them first
  if(node->_semaphores && !node->_semaphores->to_acquire.empty()) {
    SmallVector<Node*> nodes;
    if(!node->_acquire_all(nodes)) {
      _schedule(worker, nodes);
      return;
    }
    node->_state.fetch_or(Node::ACQUIRED, std::memory_order_release);
  }
 
  // 基于不同任务类型，执行node上的任务， conds是返回值
  switch(node->_handle.index()) {
    // static task
    case Node::STATIC:{
      _invoke_static_task(worker, node);
    }
    break;
 
    // dynamic task
    case Node::DYNAMIC: {
      _invoke_dynamic_task(worker, node);
    }
    break;
    ......
  }
 
  invoke_successors:
 
  // if releasing semaphores exist, release them
  if(node->_semaphores && !node->_semaphores->to_release.empty()) {
    _schedule(worker, node->_release_all());
  }
   
  // Reset the join counter to support the cyclic control flow.
  // + We must do this before scheduling the successors to avoid race
  //   condition on _dependents.
  // + We must use fetch_add instead of direct assigning
  //   because the user-space call on "invoke" may explicitly schedule
  //   this task again (e.g., pipeline) which can access the join_counter.
  if((node->_state.load(std::memory_order_relaxed) & Node::CONDITIONED)) {
    node->_join_counter.fetch_add(node->num_strong_dependents(), std::memory_order_relaxed);
  }
  else {
    node->_join_counter.fetch_add(node->num_dependents(), std::memory_order_relaxed);
  }
 
  // acquire the parent flow counter
  auto& j = (node->_parent) ? node->_parent->_join_counter :
                              node->_topology->_join_counter;
 
  // Here, we want to cache the latest successor with the highest priority
  worker._cache = nullptr;
  auto max_p = static_cast<unsigned>(TaskPriority::MAX);
 
  // Invoke the task based on the corresponding type
  switch(node->_handle.index()) {
 
    // condition and multi-condition tasks
    case Node::CONDITION:
    case Node::MULTI_CONDITION: {
      for(auto cond : conds) {
        if(cond >= 0 && static_cast<size_t>(cond) < node->_successors.size()) {
          auto s = node->_successors[cond];/           }
            worker._cache = s;
            max_p = s->_priority;
          }
          else {
            _schedule(worker, s);
          }
        }
      }
    }
    break;
 
    // 非条件的任务，即全部是强依赖，则
    default: {
      for(size_t i=0; i<node->_successors.size(); ++i) {
        if(auto s = node->_successors[i];
          s->_join_counter.fetch_sub(1, std::memory_order_acq_rel) == 1) {
          j.fetch_add(1, std::memory_order_relaxed);
          // 优先级最高的自己的本worker直接执行，否则会进入 _wsq 队列等待执行
          if(s->_priority <= max_p) {
            if(worker._cache) {
              _schedule(worker, worker._cache);
            }
            worker._cache = s;
            max_p = s->_priority;
          }
          else {
            _schedule(worker, s);
          }
        }
      }
    }
    break;
  }
 
  // 通知睡觉的worker，可以进行窃取task了
  _tear_down_invoke(worker, node);
  // 指定当前执行node，开始执行新的worker
  if(worker._cache) {
    node = worker._cache;
    goto begin_invoke;
  }
}
```

### 任务调度

```c++
// Procedure: _schedule
inline void Executor::_schedule(Worker& worker, Node* node) {
   
  // We need to fetch p before the release such that the read
  // operation is synchronized properly with other thread to
  // void data race.
  auto p = node->_priority;
 
  node->_state.fetch_or(Node::READY, std::memory_order_release);
 
  // 本地worker队列
  if(worker._executor == this) {
    // 推入worker 的wsq
    worker._wsq.push(node, p);
  // 通知等待的worker
    _notifier.notify(false);
    return;
  }
 
  // 全局队列
  {
    std::lock_guard<std::mutex> lock(_wsq_mutex);
    _wsq.push(node, p);
  }
  // 通知等待的worker
  _notifier.notify(false);
}
```

## 其他链接

1. [github 地址](https://github.com/taskflow/taskflow)
2. [2022年论文](https://tsung-wei-huang.github.io/papers/tpds21-taskflow.pdf)
3. [作者主页](https://tsung-wei-huang.github.io/team/)
