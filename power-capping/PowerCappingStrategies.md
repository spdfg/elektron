Elektron: Power-Capping Strategies
==================================

##__Power-Capping Policies__

* **Extrema** - A dynamic power-capping strategy that is able to make smart trade-offs
between makespan and power consumption. *Extrema* reacts to power trends in the
cluster and restrains the power consumption of the cluster to a power
envelope defined by a high threshold and low threshold.
* **Progressive-Extrema** - A modified version *Extrema* that performs
power-capping in phases. Unlike in *Extrema*, where picking a previously
capped node as a victim resulted in a NO-OP, *Progressive-Extrema* applies
a harsher capping value for that victim.