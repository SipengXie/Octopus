- Only scheduler_heuristic is the scheduler we actually use in octopus
    - heuristic_simple corresponds to octopus, load_balancing corresponds to quecc
    - Our scheduler can optimize these algorithms that require DAG scheduling
- There are two types of Processors, one using tree optimization and one implemented with a simple list
    - The tree optimization has a larger constant, so using a list can be considered when the graph scale is not large
- Processor_Simple does not implement IBP, it just appends, which mimics the operation of a thread pool
    - This is because DAG based on thread pools dynamically allocates processors, which can cause long-tail effects (waiting for large transactions)
- schedule_test is used to compare the scheduling capabilities of octopus, queCC, and octopus