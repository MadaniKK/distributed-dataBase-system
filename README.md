# Conflicts, Consistency, and Clocks

There is currently no known bugs in this portion of the code.

### Test cases

For physical clocks we created a simple scenario where we try to resolve three concurrent events (with the same key and physical clock), and checked whether the resolver picked out the key-value pair with the maximum value.
For vector clocks, we first tested the HappensBefore function against an edge case where two vectors have different sets of keys, but some of the keys have value 0. We then tried to resolve concurrent events and checked whether the new clock is correct and whether the resolver picked out the key-value pair with the maximum value. Finally, we checked that the local vector is updated correctly upon receiving messages with new clocks.

# Leaderless Replication

There is currently no known bugs in this portion of the code. There are edge cases that the quorum r/w cannot handle, such as concurrent read and writes, and scenarios where a write succeeds on less than W replicas.

### Test cases

We mainly did intergation testing. The first scenario we tested was eventual consistency in a cluster of 10 nodes. We then added partitioning (of 1 node) into a cluster of 3 nodes to see if eventual consistency is still satisfied.


# Partitioning

There is currently no known bugs in this portion of the code.

### Test cases
Our testing consists of testing for an edge case in lookup as well as an integration test where we add and remove a large number of replica groups. 
