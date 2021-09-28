# Raft
Raft Consensus Protocol

Test (2A): initial election ...

  ... Passed --   3.0  3   44    0
	
Test (2A): election after network failure ...

  ... Passed --   4.4  3  104    0
	
Test (2B): basic agreement ...

  ... Passed --   1.1  5   36    3
	
Test (2B): agreement despite follower disconnection ...

  ... Passed --   6.5  3  112    8
	
Test (2B): no agreement if too many followers disconnect ...

  ... Passed --   3.9  5  184    3
	
Test (2B): concurrent Start()s ...

  ... Passed --   0.8  3   10    6
	
Test (2B): rejoin of partitioned leader ...

  ... Passed --   6.6  3  154    4
	
Test (2B): leader backs up quickly over incorrect follower logs ...

  ... Passed --  31.5  5 2292  102
	
Test (2B): RPC counts aren't too high ...

  ... Passed --   2.2  3   34   12
	
Test (2C): basic persistence ...

  ... Passed --   4.4  3   86    6
	
Test (2C): more persistence ...

  ... Passed --  23.1  5 1260   16
	
Test (2C): partitioned leader and one follower crash, leader restarts ...

  ... Passed --   2.0  3   32    4
	
Test (2C): Figure 8 ...

  ... Passed --  32.5  5 1012   12
	
Test (2C): unreliable agreement ...

  ... Passed --   6.9  5  216  246
	
Test (2C): Figure 8 (unreliable) ...

  ... Passed --  43.5  5 3076  352
	
Test (2C): churn ...

  ... Passed --  16.4  5  696  340
	
Test (2C): unreliable churn ...

  ... Passed --  16.3  5  564  145
	
PASS

ok      raft    205.454s
